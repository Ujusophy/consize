package orchestrator

import (
	"context"
	"errors"
	"fmt"

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

func (s *Service) ExecuteRecommendation(ctx context.Context, recID int64, mode, actor string) (Response, error) {
	rec, err := s.st.GetRecommendation(ctx, recID)
	if err != nil {
		return Response{}, err
	}
	if rec.Status != store.RecommendationPending && rec.Status != store.RecommendationPlanned {
		return Response{}, fmt.Errorf("recommendation status is %q, not actionable", rec.Status)
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
	})
	if err != nil {
		return out, err
	}
	status := store.RecommendationPlanned
	if out.Executed != nil && out.Executed.Result == store.ActionExecuted {
		status = store.RecommendationExecuted
	}
	if err := s.st.SetRecommendationStatus(ctx, rec.ID, status); err != nil {
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
	requested, err := s.st.CreateActionEvent(ctx, store.ActionEvent{
		RecommendationID: req.RecommendationID,
		ResourceID:       req.ResourceID,
		PluginID:         req.PluginID,
		ActionType:       req.ActionType,
		Actor:            req.Actor,
		Mode:             req.Mode,
		Result:           store.ActionRequested,
		Message:          "action requested",
		Parameters:       req.Parameters,
		PolicyDecision:   decision,
	})
	if err != nil {
		return out, err
	}
	out.Requested = requested
	if decision.Decision == policy.DecisionBlocked {
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
		RecommendationID: req.RecommendationID,
		ResourceID:       req.ResourceID,
		PluginID:         req.PluginID,
		ActionType:       req.ActionType,
		Actor:            req.Actor,
		Mode:             req.Mode,
		Result:           store.ActionPlanned,
		Message:          plan.Summary,
		Parameters:       req.Parameters,
		Plan:             &plan,
		PolicyDecision:   decision,
	})
	if err != nil {
		return out, err
	}
	out.Planned = planned
	return out, nil
}
