package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/consize-oss/consize/pkg/resource"
	"golang.org/x/sys/unix"
)

type diskState struct {
	Version         int                          `json:"version"`
	Resources       map[string]resource.Resource `json:"resources"`
	Recommendations map[int64]Recommendation     `json:"recommendations"`
	Actions         map[int64]ActionEvent        `json:"actions"`
	Jobs            map[int64]Job                `json:"jobs"`
	NextActionID    int64                        `json:"next_action_id"`
	NextRecID       int64                        `json:"next_recommendation_id"`
}

// OpenDurable owns one local state file for its entire lifetime; other processes fail closed.
func OpenDurable(path string) (*Memory, error) {
	if path == "" {
		return nil, errors.New("state_path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		return nil, fmt.Errorf("state store already owned: %w", err)
	}
	m := NewMemory()
	m.path = path
	m.lockFile = lock
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := m.persistLocked(); err != nil {
			m.Close()
			return nil, err
		}
		return m, nil
	}
	if err != nil {
		m.Close()
		return nil, err
	}
	var state diskState
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&state); err != nil {
		m.Close()
		return nil, fmt.Errorf("invalid state file: %w", err)
	}
	if state.Version != 1 || state.Resources == nil || state.Recommendations == nil || state.Actions == nil || state.Jobs == nil || state.NextActionID < 1 || state.NextRecID < 1 {
		m.Close()
		return nil, errors.New("invalid state schema")
	}
	if err := validateState(state); err != nil {
		m.Close()
		return nil, err
	}
	m.resources = state.Resources
	m.recs = state.Recommendations
	m.actions = state.Actions
	m.jobs = state.Jobs
	m.nextActionID = state.NextActionID
	m.nextRecID = state.NextRecID
	return m, nil
}

func validateState(state diskState) error {
	for id, res := range state.Resources {
		if id == "" || res.ID != id || res.Type == "" {
			return errors.New("invalid resource identity in durable state")
		}
	}
	for id, rec := range state.Recommendations {
		if id < 1 || rec.ID != id || id >= state.NextRecID {
			return errors.New("invalid recommendation sequence in durable state")
		}
		if _, ok := state.Resources[rec.ResourceID]; !ok {
			return errors.New("missing recommendation resource in durable state")
		}
	}
	for id, event := range state.Actions {
		if id < 1 || event.ID != id || id >= state.NextActionID {
			return errors.New("invalid action sequence in durable state")
		}
		if event.RecommendationID > 0 {
			if _, ok := state.Recommendations[event.RecommendationID]; !ok {
				return errors.New("missing audit recommendation in durable state")
			}
		}
	}
	for id, job := range state.Jobs {
		rec, ok := state.Recommendations[id]
		if !ok || job.ID != id || job.Resource.ID != rec.ResourceID || job.Plan.ResourceID != rec.ResourceID || job.Plan.PluginID != rec.PluginID || job.Actor == "" {
			return errors.New("invalid safety job identity in durable state")
		}
		switch job.State {
		case "prepared", "applying", "waiting_rollout", "verifying", "rollback_pending", "rolling_back", "rollback_verifying", "verified", "rolled_back", "manual_intervention", "cancelled":
		default:
			return errors.New("unknown safety state")
		}
		if (job.State == "verifying" || job.State == "rollback_verifying") && job.WindowStart.IsZero() {
			return errors.New("missing observation window in durable state")
		}
		if !Terminal(job.State) && job.Deadline.IsZero() {
			return errors.New("missing action deadline in durable state")
		}
	}
	return nil
}

func (m *Memory) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lockFile == nil {
		return nil
	}
	err := m.lockFile.Close()
	m.lockFile = nil
	m.poison = errors.New("store closed")
	return err
}

func (m *Memory) Durable() bool { return m.path != "" }

func (m *Memory) persistLocked() (err error) {
	if m.poison != nil {
		return m.poison
	}
	if m.path == "" {
		return nil
	}
	defer func() {
		if err != nil {
			m.poison = fmt.Errorf("durable store unavailable: %w", err)
		}
	}()
	state := diskState{1, m.resources, m.recs, m.actions, m.jobs, m.nextActionID, m.nextRecID}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(m.path), ".consize-state-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(file.Name(), m.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(m.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func clone[T any](value T) T {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var out T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&out); err != nil {
		panic(err)
	}
	return out
}
