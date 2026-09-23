package store

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/consize-oss/consize/internal/bootstrap"
	"github.com/consize-oss/consize/internal/policy"
	"github.com/consize-oss/consize/internal/verification"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
)

type Job struct {
	Result       *verification.Result         `json:"verification_result,omitempty"`
	ID           int64                        `json:"id"`
	Resource     resource.Resource            `json:"resource"`
	Plan         plugin.ActionPlan            `json:"plan"`
	Baseline     plugin.MetricsSnapshot       `json:"baseline"`
	Verification bootstrap.VerificationConfig `json:"verification"`
	Policy       policy.Decision              `json:"policy"`
	Actor        string                       `json:"actor"`
	State        string                       `json:"state"`
	Attempts     int                          `json:"attempts"`
	NextRun      time.Time                    `json:"next_run"`
	WindowStart  time.Time                    `json:"window_start"`
	Deadline     time.Time                    `json:"deadline"`
	LastError    string                       `json:"last_error,omitempty"`
	UpdatedAt    time.Time                    `json:"updated_at"`
}

func Terminal(state string) bool {
	return state == "verified" || state == "rolled_back" || state == "manual_intervention" || state == "cancelled"
}

type JobStore interface {
	Store
	Durable() bool
	CreateJob(context.Context, Job) (Job, error)
	SaveJob(context.Context, Job, string) error
	ListJobs(context.Context) ([]Job, error)
}

func (m *Memory) CreateJob(_ context.Context, job Job) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.poison != nil {
		return Job{}, m.poison
	}
	if existing, ok := m.jobs[job.ID]; ok {
		return clone(existing), nil
	}
	rec, ok := m.recs[job.ID]
	if !ok {
		return Job{}, ErrNotFound
	}
	if rec.Status != RecommendationPending && rec.Status != RecommendationPlanned {
		return Job{}, errors.New("recommendation is not actionable")
	}
	for _, existing := range m.jobs {
		if existing.Resource.ID == job.Resource.ID && (!Terminal(existing.State) || existing.State == "manual_intervention") {
			return Job{}, errors.New("resource already has an active or unresolved action")
		}
	}
	job.State = "prepared"
	job.UpdatedAt = m.now().UTC()
	job.NextRun = job.UpdatedAt
	m.jobs[job.ID] = clone(job)
	m.jobEventLocked(job, "Durable action prepared before mutation")
	return clone(job), m.persistLocked()
}

func (m *Memory) SaveJob(_ context.Context, job Job, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.poison != nil {
		return m.poison
	}
	if _, ok := m.jobs[job.ID]; !ok {
		return ErrNotFound
	}
	job.UpdatedAt = m.now().UTC()
	m.jobs[job.ID] = clone(job)
	m.jobEventLocked(job, message)
	return m.persistLocked()
}

func (m *Memory) jobEventLocked(job Job, message string) {
	event := ActionEvent{ID: m.nextActionID, RecommendationID: job.ID, ResourceID: job.Resource.ID, PluginID: job.Plan.PluginID, Actor: job.Actor, Mode: "approved", Result: job.State, Message: message, PolicyDecision: job.Policy, CreatedAt: m.now().UTC(), Plan: &job.Plan}
	event.VerificationResult = job.Result
	m.actions[event.ID] = clone(event)
	m.nextActionID++
	rec := m.recs[job.ID]
	rec.Status = job.State
	rec.PolicyID = job.Policy.PolicyID
	rec.UpdatedAt = m.now().UTC()
	m.recs[job.ID] = rec
	if job.State == "waiting_rollout" || job.State == "verified" || job.State == "rollback_verifying" || job.State == "rolled_back" {
		res := m.resources[job.Resource.ID]
		if job.State == "rolled_back" || job.State == "rollback_verifying" {
			res.CurrentState = clone(job.Plan.OriginalState)
		} else {
			res.CurrentState = clone(job.Plan.AppliedState)
		}
		m.resources[res.ID] = res
	}
}

func (m *Memory) ListJobs(context.Context) ([]Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.poison != nil {
		return nil, m.poison
	}
	out := make([]Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		out = append(out, clone(job))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
