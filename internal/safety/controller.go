package safety

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/consize-oss/consize/internal/bootstrap"
	"github.com/consize-oss/consize/internal/policy"
	"github.com/consize-oss/consize/internal/recommender"
	"github.com/consize-oss/consize/internal/store"
	"github.com/consize-oss/consize/internal/verification"
	"github.com/consize-oss/consize/pkg/plugin"
)

type Controller struct {
	recommendationCfg bootstrap.RecommenderConfig
	mu                sync.Mutex
	st                store.JobStore
	plugins           *plugin.Manager
	policies          *policy.Engine
	cfg               bootstrap.VerificationConfig
	now               func() time.Time
}

func New(st store.JobStore, plugins *plugin.Manager, policies *policy.Engine, cfg bootstrap.VerificationConfig, recommendationConfig ...bootstrap.RecommenderConfig) *Controller {
	var rc bootstrap.RecommenderConfig
	if len(recommendationConfig) > 0 {
		rc = recommendationConfig[0]
	}
	return &Controller{st: st, plugins: plugins, policies: policies, cfg: cfg, now: time.Now, recommendationCfg: rc}
}

func durations(cfg bootstrap.VerificationConfig) (time.Duration, time.Duration, time.Duration, error) {
	if err := verification.ValidateChecks(cfg); err != nil {
		return 0, 0, 0, err
	}
	wait, err := time.ParseDuration(cfg.Wait)
	if err != nil || wait < 5*time.Minute {
		return 0, 0, 0, errors.New("verification wait must be at least 5m")
	}
	isolation := 30 * time.Minute
	if cfg.Isolation != "" {
		isolation, err = time.ParseDuration(cfg.Isolation)
		if err != nil || isolation < 30*time.Minute {
			return 0, 0, 0, errors.New("verification isolation must be at least 30m for rate and restart lookbacks")
		}
	}
	timeout := 10 * time.Minute
	if cfg.Timeout != "" {
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil || timeout < time.Minute {
			return 0, 0, 0, errors.New("verification timeout must be at least 1m")
		}
	}
	if cfg.MaxMemoryP95IncreaseRatio < 1 || cfg.MaxCPUP95IncreaseRatio < 1 || cfg.MaxRestartIncrease < 0 || math.IsNaN(cfg.MaxMemoryP95IncreaseRatio) || math.IsNaN(cfg.MaxCPUP95IncreaseRatio) || math.IsInf(cfg.MaxMemoryP95IncreaseRatio, 0) || math.IsInf(cfg.MaxCPUP95IncreaseRatio, 0) {
		return 0, 0, 0, errors.New("invalid verification thresholds")
	}
	return wait, isolation, timeout, nil
}

func (c *Controller) Submit(ctx context.Context, id int64, actor string) (store.Job, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.st.Durable() {
		return store.Job{}, errors.New("execution requires a durable state store")
	}
	if err := c.st.Health(ctx); err != nil {
		return store.Job{}, err
	}
	jobs, err := c.st.ListJobs(ctx)
	if err != nil {
		return store.Job{}, err
	}
	for _, job := range jobs {
		if job.ID == id {
			return job, nil
		}
	}
	if actor == "" {
		return store.Job{}, errors.New("an explicit actor is required")
	}
	if !c.cfg.Enabled {
		return store.Job{}, errors.New("metrics verification must be enabled")
	}
	if _, _, _, err := durations(c.cfg); err != nil {
		return store.Job{}, err
	}
	rec, err := c.st.GetRecommendation(ctx, id)
	if err != nil {
		return store.Job{}, err
	}
	if rec.Confidence != "medium" && rec.Confidence != "high" {
		return store.Job{}, errors.New("insufficient recommendation confidence for execution")
	}
	res, err := c.st.GetResource(ctx, rec.ResourceID)
	if err != nil {
		return store.Job{}, err
	}
	action, err := c.plugins.ActionPlugin(rec.PluginID, res.Type, rec.ActionType)
	if err != nil {
		return store.Job{}, err
	}
	recoverable, ok := action.(plugin.RecoverableActionPlugin)
	if !ok {
		return store.Job{}, errors.New("action plugin must implement inspection, readiness and rollback")
	}
	decision := c.policies.Evaluate(ctx, res, action.Manifest(), "approved", actor)
	if decision.Decision == policy.DecisionBlocked || decision.Decision == policy.DecisionRecommendOnly {
		return store.Job{}, errors.New("policy blocked execution")
	}
	plan, err := action.Plan(ctx, plugin.ActionInput{Resource: res, ActionType: rec.ActionType, Mode: "approved", Actor: actor, Parameters: rec.Parameters})
	if err != nil {
		return store.Job{}, err
	}
	if !plan.RollbackAvailable {
		return store.Job{}, errors.New("action does not provide a recoverable original state")
	}
	plan.PluginVersion = action.Manifest().Version
	if artifact, ok := action.(plugin.ArtifactIdentity); ok {
		plan.PluginArtifactDigest = artifact.ArtifactDigest()
	}
	if plan.PluginID != action.ID() || plan.ResourceID != res.ID || plan.ActionType != rec.ActionType {
		return store.Job{}, errors.New("action plugin returned a mismatched plan")
	}
	if plan.OriginalState == nil || plan.AppliedState == nil {
		return store.Job{}, errors.New("action plan must provide original and applied registry state")
	}
	preflight, ok := action.(plugin.PreflightActionPlugin)
	if !ok {
		return store.Job{}, errors.New("action plugin must implement live preflight")
	}
	plan.Preflight, err = preflight.Preflight(ctx, plan)
	if err != nil {
		return store.Job{}, err
	}
	if err = plugin.RequirePreflight(plan.Preflight); err != nil {
		return store.Job{}, err
	}
	if _, err := json.Marshal(plan); err != nil {
		return store.Job{}, fmt.Errorf("invalid action plan: %w", err)
	}
	if err := recoverable.Ready(ctx, plan); err != nil {
		return store.Job{}, fmt.Errorf("baseline workload is not healthy: %w", err)
	}
	metrics, err := c.plugins.MetricsPlugin(c.cfg.MetricsPluginID, res.Type)
	if err != nil {
		return store.Job{}, err
	}
	if _, ok := metrics.(plugin.WindowedMetricsPlugin); !ok {
		return store.Job{}, errors.New("metrics plugin must support explicit time windows")
	}
	baseline, err := metrics.ReadMetrics(ctx, res)
	if err != nil {
		return store.Job{}, err
	}
	for _, key := range verification.RequiredSignals(c.cfg) {
		if _, ok := baseline.Coverage[key]; !ok {
			return store.Job{}, errors.New("metrics plugin must supply timestamped coverage for required signals")
		}
	}
	if baseline.ResourceID != res.ID || baseline.CollectedAt.IsZero() || c.now().Sub(baseline.CollectedAt) > 2*time.Minute || baseline.CollectedAt.After(c.now().Add(time.Minute)) {
		return store.Job{}, errors.New("baseline evidence is stale or mismatched")
	}
	if rec.AlgorithmID == recommender.BuiltInHeadroomAlgorithmID {
		assessment := recommender.AssessConfidence(c.recommendationCfg, res, baseline, c.now())
		if assessment.Label == "low" {
			return store.Job{}, fmt.Errorf("recommendation history rejected: %v", assessment.Reasons)
		}
	}
	result := verification.New(c.cfg).Verify(baseline, baseline)
	if result.Status != verification.StatusPassed {
		return store.Job{}, fmt.Errorf("baseline metrics rejected: %v", result.Reasons)
	}
	if _, err := json.Marshal(baseline); err != nil {
		return store.Job{}, fmt.Errorf("invalid baseline: %w", err)
	}
	return c.st.CreateJob(ctx, store.Job{ID: id, Resource: res, Plan: plan, Baseline: baseline, Verification: c.cfg, Policy: decision, Actor: actor, Deadline: c.now().Add(10 * time.Minute)})
}

func (c *Controller) Run(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := c.Tick(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// Recover requires a new operator approval and still refuses external drift.
func (c *Controller) Recover(ctx context.Context, id int64, actor string) (store.Job, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if actor == "" {
		return store.Job{}, errors.New("recovery requires an explicit actor")
	}
	jobs, err := c.st.ListJobs(ctx)
	if err != nil {
		return store.Job{}, err
	}
	for _, job := range jobs {
		if ctx.Err() != nil {
			return store.Job{}, ctx.Err()
		}
		if job.ID != id {
			continue
		}
		if job.State != "manual_intervention" {
			return store.Job{}, errors.New("only unresolved actions can be recovered")
		}
		action, err := c.plugins.ActionPlugin(job.Plan.PluginID, job.Resource.Type, job.Plan.ActionType)
		if err != nil {
			return store.Job{}, err
		}
		if err := matchingRelease(job.Plan, action); err != nil {
			return store.Job{}, err
		}
		recovery, ok := action.(plugin.RecoverableActionPlugin)
		if !ok {
			return store.Job{}, errors.New("recovery plugin unavailable")
		}
		state, err := recovery.Inspect(ctx, job.Plan)
		if err != nil {
			return store.Job{}, err
		}
		if state != plugin.StateOriginal && state != plugin.StateApplied {
			return store.Job{}, errors.New("external drift must be reconciled by the operator before retrying recovery")
		}
		decision := c.policies.Evaluate(ctx, job.Resource, action.Manifest(), "approved", actor)
		if decision.Decision == policy.DecisionBlocked || decision.Decision == policy.DecisionRecommendOnly {
			return store.Job{}, errors.New("policy blocked recovery")
		}
		job.Policy = decision
		job.Actor = actor
		job.Deadline = c.now().Add(10 * time.Minute)
		if err := c.save(ctx, job, "rollback_pending", "Operator approved retry of captured-state restoration"); err != nil {
			return store.Job{}, err
		}
		job.State = "rollback_pending"
		return job, nil
	}
	return store.Job{}, store.ErrNotFound
}

func (c *Controller) Tick(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ctx.Err() != nil {
		return nil
	}
	if err := c.st.Health(ctx); err != nil {
		return err
	}
	jobs, err := c.st.ListJobs(ctx)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if store.Terminal(job.State) || c.now().Before(job.NextRun) {
			continue
		}
		jobCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := c.step(jobCtx, job)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) save(ctx context.Context, job store.Job, state, message string) error {
	job.State = state
	job.LastError = ""
	if state == "manual_intervention" {
		job.LastError = message
	}
	job.Attempts = 0
	job.NextRun = c.now().Add(5 * time.Second)
	return c.st.SaveJob(ctx, job, message)
}

func (c *Controller) retry(ctx context.Context, job store.Job, err error) error {
	job.Attempts++
	job.LastError = err.Error()
	job.NextRun = c.now().Add(time.Duration(min(job.Attempts, 12)) * 5 * time.Second)
	if !job.Deadline.IsZero() && !c.now().Before(job.Deadline) {
		if job.State == "verifying" || job.State == "waiting_rollout" || job.State == "applying" {
			return c.fail(ctx, job, "timeout: "+err.Error(), job.Verification.RollbackOnTimeout)
		}
		return c.save(ctx, job, "manual_intervention", "Recovery deadline exceeded: "+err.Error())
	}
	return c.st.SaveJob(ctx, job, "Retry scheduled: "+err.Error())
}

func (c *Controller) fail(ctx context.Context, job store.Job, message string, rollback bool) error {
	if !rollback {
		return c.save(ctx, job, "manual_intervention", message+"; policy requires manual recovery")
	}
	job.Deadline = c.now().Add(10 * time.Minute)
	return c.save(ctx, job, "rollback_pending", message+"; rollback authorized by captured policy")
}

func (c *Controller) beginWindow(ctx context.Context, job store.Job, rollback bool) error {
	wait, isolation, timeout, err := durations(job.Verification)
	if err != nil {
		return c.save(ctx, job, "manual_intervention", err.Error())
	}
	job.WindowStart = c.now().Add(isolation)
	job.Deadline = job.WindowStart.Add(wait + timeout)
	state := "verifying"
	if rollback {
		state = "rollback_verifying"
	}
	return c.save(ctx, job, state, "Healthy rollout confirmed; isolated metrics verification scheduled")
}

func matchingRelease(plan plugin.ActionPlan, action plugin.ActionPlugin) error {
	if plan.PluginVersion != "" && plan.PluginVersion != action.Manifest().Version {
		return errors.New("action plugin version changed; restore captured release before recovery")
	}
	if plan.PluginArtifactDigest != "" {
		artifact, ok := action.(plugin.ArtifactIdentity)
		if !ok || artifact.ArtifactDigest() != plan.PluginArtifactDigest {
			return errors.New("action plugin artifact changed; restore captured binary before recovery")
		}
	}
	return nil
}

func (c *Controller) step(ctx context.Context, job store.Job) error {
	action, err := c.plugins.ActionPlugin(job.Plan.PluginID, job.Resource.Type, job.Plan.ActionType)
	if err != nil {
		return c.retry(ctx, job, err)
	}
	if err := matchingRelease(job.Plan, action); err != nil {
		return c.save(ctx, job, "manual_intervention", err.Error())
	}
	recovery, ok := action.(plugin.RecoverableActionPlugin)
	if !ok {
		return c.retry(ctx, job, errors.New("recovery plugin unavailable"))
	}
	state, err := recovery.Inspect(ctx, job.Plan)
	if err != nil {
		return c.retry(ctx, job, err)
	}
	if state == plugin.StateDrifted {
		return c.save(ctx, job, "manual_intervention", "External resource drift detected; no changes overwritten")
	}
	if state != plugin.StateOriginal && state != plugin.StateApplied {
		return c.save(ctx, job, "manual_intervention", "Plugin returned an unknown resource state")
	}
	if !job.Deadline.IsZero() && !c.now().Before(job.Deadline) {
		if job.State == "rollback_pending" || job.State == "rolling_back" || job.State == "rollback_verifying" {
			return c.save(ctx, job, "manual_intervention", "Rollback deadline exceeded; recovery requires operator attention")
		}
		return c.fail(ctx, job, "Action or verification deadline exceeded", job.Verification.RollbackOnTimeout)
	}
	switch job.State {
	case "prepared":
		job.Deadline = c.now().Add(5 * time.Minute)
		return c.save(ctx, job, "applying", "Apply intent persisted before infrastructure mutation")
	case "applying":
		if state == plugin.StateOriginal {
			preflight, ok := action.(plugin.PreflightActionPlugin)
			if !ok {
				return c.save(ctx, job, "cancelled", "Action plugin cannot perform live preflight; no mutation performed")
			}
			checks, err := preflight.Preflight(ctx, job.Plan)
			if err != nil {
				return c.retry(ctx, job, err)
			}
			job.Plan.Preflight = checks
			if err = plugin.RequirePreflight(checks); err != nil {
				return c.save(ctx, job, "cancelled", err.Error()+"; no mutation performed")
			}
			if _, err := action.Execute(ctx, job.Plan); err != nil {
				var blocked *plugin.PreflightError
				if errors.As(err, &blocked) {
					job.Plan.Preflight = blocked.Checks
					actual, inspectErr := recovery.Inspect(ctx, job.Plan)
					if inspectErr == nil && actual == plugin.StateOriginal {
						return c.save(ctx, job, "cancelled", err.Error()+"; no mutation performed")
					}
				}
				return c.retry(ctx, job, err)
			}
		}
		observed, err := recovery.Inspect(ctx, job.Plan)
		if err != nil {
			return c.retry(ctx, job, err)
		}
		if observed != plugin.StateApplied {
			return c.retry(ctx, job, errors.New("apply did not produce the prepared target state"))
		}
		return c.save(ctx, job, "waiting_rollout", "Apply reconciled; waiting for workload readiness")
	case "waiting_rollout":
		if state != plugin.StateApplied {
			return c.fail(ctx, job, "Applied state no longer present", job.Verification.RollbackOnFailure)
		}
		if err := recovery.Ready(ctx, job.Plan); err != nil {
			return c.retry(ctx, job, err)
		}
		return c.beginWindow(ctx, job, false)
	case "verifying", "rollback_verifying":
		rollback := job.State == "rollback_verifying"
		expected := plugin.StateApplied
		if rollback {
			expected = plugin.StateOriginal
		}
		if state != expected {
			return c.save(ctx, job, "manual_intervention", "Resource changed during verification")
		}
		if err := recovery.Ready(ctx, job.Plan); err != nil {
			if !rollback {
				return c.fail(ctx, job, "Workload readiness lost: "+err.Error(), job.Verification.RollbackOnFailure)
			}
			return c.retry(ctx, job, err)
		}
		wait, _, _, err := durations(job.Verification)
		if err != nil {
			return c.save(ctx, job, "manual_intervention", err.Error())
		}
		if c.now().Before(job.WindowStart.Add(wait)) {
			return nil
		}
		metrics, err := c.plugins.MetricsPlugin(job.Verification.MetricsPluginID, job.Resource.Type)
		if err != nil {
			return c.retry(ctx, job, err)
		}
		windowed, ok := metrics.(plugin.WindowedMetricsPlugin)
		if !ok {
			return c.retry(ctx, job, errors.New("metrics plugin cannot collect isolated window"))
		}
		after, err := windowed.ReadMetricsBetween(ctx, job.Resource, job.WindowStart, c.now())
		if err != nil {
			return c.retry(ctx, job, err)
		}
		if after.ResourceID != job.Resource.ID || after.CollectedAt.Before(job.WindowStart) {
			return c.retry(ctx, job, errors.New("verification evidence does not match resource or time window"))
		}
		for _, key := range verification.RequiredSignals(job.Verification) {
			coverage, ok := after.Coverage[key]
			if !ok || coverage.FirstSample.Before(job.WindowStart) || coverage.LastSample.After(c.now()) {
				return c.retry(ctx, job, errors.New("verification evidence lacks isolated timestamped coverage"))
			}
		}
		result := verification.New(job.Verification).Verify(job.Baseline, after)
		job.Result = &result
		if result.Status == verification.StatusInconclusive {
			return c.retry(ctx, job, errors.New("verification evidence incomplete"))
		}
		if result.Status == verification.StatusFailed {
			if rollback {
				return c.save(ctx, job, "manual_intervention", fmt.Sprintf("Rollback state restored but health checks failed: %v", result.Reasons))
			}
			return c.fail(ctx, job, fmt.Sprintf("Health checks failed: %v", result.Reasons), job.Verification.RollbackOnFailure)
		}
		job.LastError = ""
		terminal := "verified"
		if rollback {
			terminal = "rolled_back"
		}
		return c.save(ctx, job, terminal, "Resource state, readiness and isolated metrics checks passed")
	case "rollback_pending":
		return c.save(ctx, job, "rolling_back", "Rollback intent persisted before restoration")
	case "rolling_back":
		if state == plugin.StateApplied {
			if _, err := recovery.Rollback(ctx, job.Plan); err != nil {
				return c.retry(ctx, job, err)
			}
		}
		observed, err := recovery.Inspect(ctx, job.Plan)
		if err != nil {
			return c.retry(ctx, job, err)
		}
		if observed != plugin.StateOriginal {
			return c.retry(ctx, job, errors.New("rollback did not restore the captured state"))
		}
		if err := recovery.Ready(ctx, job.Plan); err != nil {
			return c.retry(ctx, job, err)
		}
		return c.beginWindow(ctx, job, true)
	default:
		return c.save(ctx, job, "manual_intervention", "Unknown action state; execution stopped")
	}
}
