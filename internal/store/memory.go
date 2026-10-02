package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/consize-oss/consize/pkg/resource"
)

var ErrNotFound = errors.New("not found")

type Memory struct {
	path         string
	lockFile     *os.File
	poison       error
	jobs         map[int64]Job
	mu           sync.RWMutex
	now          func() time.Time
	resources    map[string]resource.Resource
	recs         map[int64]Recommendation
	actions      map[int64]ActionEvent
	nextActionID int64
	nextRecID    int64
}

func NewMemory() *Memory {
	return &Memory{
		jobs:         map[int64]Job{},
		now:          time.Now,
		resources:    map[string]resource.Resource{},
		recs:         map[int64]Recommendation{},
		actions:      map[int64]ActionEvent{},
		nextActionID: 1,
		nextRecID:    1,
	}
}

func (m *Memory) Health(context.Context) error { m.mu.RLock(); defer m.mu.RUnlock(); return m.poison }

func (m *Memory) UpsertResource(_ context.Context, res resource.Resource) (resource.Resource, error) {
	now := m.now().UTC()
	normalized, err := res.Normalize(now)
	if err != nil {
		return resource.Resource{}, fmt.Errorf("normalize resource: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, existing := range m.resources {
		if id == normalized.ID || !resource.SameIdentity(existing, normalized) {
			continue
		}
		if !resource.IsCanonicalID(id) && resource.IsCanonicalID(normalized.ID) {
			// Preserve references created before canonical IDs were introduced.
			normalized.ID = id
			break
		}
		return resource.Resource{}, fmt.Errorf("%w: identity already registered as %q", resource.ErrIdentityConflict, id)
	}

	if existing, ok := m.resources[normalized.ID]; ok {
		if !resource.SameIdentity(existing, normalized) {
			return resource.Resource{}, fmt.Errorf("%w: id %q belongs to a different provider resource", resource.ErrIdentityConflict, normalized.ID)
		}
		if normalized.ObservedAt.Before(existing.LastSeenAt) {
			return resource.Resource{}, fmt.Errorf("stale resource observation: observed_at %s precedes last_seen_at %s", normalized.ObservedAt, existing.LastSeenAt)
		}
		if !resource.ValidLifecycleTransition(existing.LifecycleState, normalized.LifecycleState) {
			return resource.Resource{}, fmt.Errorf("invalid resource lifecycle transition from %q to %q", existing.LifecycleState, normalized.LifecycleState)
		}
		normalized.FirstSeenAt = existing.FirstSeenAt
		normalized.CreatedAt = existing.CreatedAt
	}
	normalized.LastSeenAt = normalized.ObservedAt
	normalized.UpdatedAt = now
	if err := normalized.Validate(); err != nil {
		return resource.Resource{}, fmt.Errorf("validate resource update: %w", err)
	}
	m.resources[normalized.ID] = clone(normalized)
	return clone(normalized), m.persistLocked()
}

func (m *Memory) GetResource(_ context.Context, id string) (resource.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res, ok := m.resources[id]
	if !ok {
		return resource.Resource{}, ErrNotFound
	}
	return clone(res), nil
}

func (m *Memory) ListResources(context.Context) ([]resource.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]resource.Resource, 0, len(m.resources))
	for _, res := range m.resources {
		out = append(out, res)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return clone(out), nil
}

func (m *Memory) CreateRecommendation(_ context.Context, rec Recommendation) (Recommendation, error) {
	if rec.ResourceID == "" || rec.PluginID == "" || rec.ActionType == "" {
		return Recommendation{}, errors.New("resource_id, plugin_id, and action_type are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now().UTC()
	rec.ID = m.nextRecID
	m.nextRecID++
	if rec.Current == nil {
		rec.Current = map[string]any{}
	}
	if rec.Proposed == nil {
		rec.Proposed = map[string]any{}
	}
	if rec.Parameters == nil {
		rec.Parameters = map[string]any{}
	}
	if rec.Status == "" {
		rec.Status = RecommendationPending
	}
	rec.CreatedAt = now
	rec.UpdatedAt = now
	m.recs[rec.ID] = clone(rec)
	return clone(rec), m.persistLocked()
}

func (m *Memory) GetRecommendation(_ context.Context, id int64) (Recommendation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.recs[id]
	if !ok {
		return Recommendation{}, ErrNotFound
	}
	return clone(rec), nil
}

func (m *Memory) ListRecommendations(context.Context) ([]Recommendation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Recommendation, 0, len(m.recs))
	for _, rec := range m.recs {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return clone(out), nil
}

func (m *Memory) SetRecommendationStatus(_ context.Context, id int64, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[id]
	if !ok {
		return ErrNotFound
	}
	rec.Status = status
	rec.UpdatedAt = m.now().UTC()
	m.recs[id] = rec
	return m.persistLocked()
}

func (m *Memory) CreateActionEvent(_ context.Context, event ActionEvent) (ActionEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	event.ID = m.nextActionID
	m.nextActionID++
	event.CreatedAt = m.now().UTC()
	if event.Parameters == nil {
		event.Parameters = map[string]any{}
	}
	m.actions[event.ID] = clone(event)
	return clone(event), m.persistLocked()
}

func (m *Memory) ListActionEvents(context.Context) ([]ActionEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ActionEvent, 0, len(m.actions))
	for _, event := range m.actions {
		out = append(out, event)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return clone(out), nil
}
