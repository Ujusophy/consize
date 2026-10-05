package orchestrator

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/consize-oss/consize/internal/policy"
	"github.com/consize-oss/consize/internal/store"
	"github.com/consize-oss/consize/pkg/plugin"
)

type Request struct {
	RecommendationID int64          `json:"recommendation_id,omitempty"`
	ResourceID       string         `json:"resource_id"`
	PluginID         string         `json:"plugin_id"`
	ActionType       string         `json:"action_type"`
	Mode             string         `json:"mode"`
	Actor            string         `json:"actor"`
	Parameters       map[string]any `json:"parameters"`
	IdempotencyKey   string         `json:"idempotency_key"`
}

type Response struct {
	Requested store.ActionEvent  `json:"requested"`
	Planned   store.ActionEvent  `json:"planned"`
	Executed  *store.ActionEvent `json:"executed,omitempty"`
}

type Service struct {
	st       store.Store
	plugins  *plugin.Manager
	policies *policy.Engine
}

func New(st store.Store, plugins *plugin.Manager, policies *policy.Engine) *Service {
	return &Service{st: st, plugins: plugins, policies: policies}
}

func (s *Service) ExecuteRecommendation(ctx context.Context, recID int64, mode, actor string, idempotencyKey ...string) (Response, error) {
	rec, err := s.st.GetRecommendation(ctx, recID)
	if err != nil {
		return Response{}, err
	}
	if rec.Status != store.RecommendationPending && rec.Status != store.RecommendationPlanned {
		return Response{}, fmt.Errorf("recommendation status is %q, not actionable", rec.Status)
	}
	if !rec.ExpiresAt.IsZero() && !rec.ExpiresAt.After(time.Now().UTC()) {
		return Response{}, store.ErrRecommendationGone
	}
	key := ""
	if len(idempotencyKey) > 0 {
		key = idempotencyKey[0]
	}
	if key == "" {
		key = fmt.Sprintf("plan-%x", sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s", rec.ID, mode, actor))))
	}
	params := rec.Parameters
	if params == nil {
		params = map[string]any{}
	}
	if _, ok := params["proposed"]; !ok && rec.Proposed != nil {
		params["proposed"] = rec.Proposed
	}
	out, err := s.Execute(ctx, Request{
		RecommendationID: rec.ID,
		ResourceID:       rec.ResourceID,
		PluginID:         rec.PluginID,
		ActionType:       rec.ActionType,
		Mode:             mode,
		Actor:            actor,
		Parameters:       params,
		IdempotencyKey:   key,
	})
	if err != nil {
		return out, err
	}
	if err := s.st.TransitionRecommendation(ctx, rec.ID, store.RecommendationPlanned, "action plan created"); err != nil && !errors.Is(err, store.ErrInvalidTransition) {
		return out, err
	}
	return out, nil
}

func (s *Service) Execute(ctx context.Context, req Request) (Response, error) {
	var out Response
	if req.Mode != "dry_run" {
		return out, errors.New("mutation must use the durable safety controller")
	}
	if req.Parameters == nil {
		req.Parameters = map[string]any{}
	}
	if err := s.st.Health(ctx); err != nil {
		return out, fmt.Errorf("store unhealthy: %w", err)
	}
	res, err := s.st.GetResource(ctx, req.ResourceID)
	if err != nil {
		return out, err
	}
	actionPlugin, err := s.plugins.ActionPlugin(req.PluginID, res.Type, req.ActionType)
	if err != nil {
		return out, err
	}
	decision := s.policies.Evaluate(ctx, res, actionPlugin.Manifest(), req.Mode, req.Actor)
	action, err := s.st.CreateAction(ctx, store.Action{
		RecommendationID: req.RecommendationID,
		ResourceID:       req.ResourceID,
		RemediationPath:  "direct_apply",
		PluginID:         req.PluginID,
		PluginVersion:    actionPlugin.Manifest().Version,
		ActionType:       req.ActionType,
		Mode:             req.Mode,
		IdempotencyKey:   req.IdempotencyKey,
		RequestedBy:      req.Actor,
		PolicyDecision:   decision,
		Parameters:       req.Parameters,
	})
	if err != nil {
		return out, err
	}
	requested, err := s.st.CreateActionEvent(ctx, store.ActionEvent{
		ActionID:         action.ID,
		RecommendationID: req.RecommendationID,
		ResourceID:       req.ResourceID,
		PluginID:         req.PluginID,
		ActionType:       req.ActionType,
		Actor:            req.Actor,
		Mode:             req.Mode,
		Result:           string(store.ActionRequested),
		Message:          "action requested",
		Parameters:       req.Parameters,
		PolicyDecision:   decision,
	})
	if err != nil {
		return out, err
	}
	out.Requested = requested
	if decision.Decision == policy.DecisionBlocked {
		_, _ = s.st.TransitionAction(ctx, action.ID, store.ActionFailed, func(a *store.Action) error {
			a.FailureCode = "policy_blocked"
			a.FailureMessage = "policy blocked action"
			return nil
		})
		return out, errors.New("policy blocked action")
	}
	plan, err := actionPlugin.Plan(ctx, plugin.ActionInput{
		Resource:   res,
		ActionType: req.ActionType,
		Mode:       req.Mode,
		Actor:      req.Actor,
		Parameters: req.Parameters,
	})
	if err != nil {
		return out, err
	}
	planned, err := s.st.CreateActionEvent(ctx, store.ActionEvent{
		ActionID:         action.ID,
		RecommendationID: req.RecommendationID,
		ResourceID:       req.ResourceID,
		PluginID:         req.PluginID,
		ActionType:       req.ActionType,
		Actor:            req.Actor,
		Mode:             req.Mode,
		Result:           string(store.ActionPlanned),
		Message:          plan.Summary,
		Parameters:       req.Parameters,
		Plan:             &plan,
		PolicyDecision:   decision,
	})
	if err != nil {
		return out, err
	}
	if _, err := s.st.TransitionAction(ctx, action.ID, store.ActionPlanned, func(a *store.Action) error {
		a.PlanResult = &store.PlanResult{Plan: plan, CreatedAt: time.Now().UTC()}
		return nil
	}); err != nil {
		return out, err
	}
	out.Planned = planned
	return out, nil
}
