package store

import (
	"context"
	"time"

	"github.com/consize-oss/consize/internal/policy"
	"github.com/consize-oss/consize/internal/verification"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
)

const (
	ActionRequested = "requested"
	ActionPlanned   = "planned"
	ActionExecuted  = "executed"
	ActionFailed    = "failed"
)

const (
	RecommendationPending  = "pending"
	RecommendationPlanned  = "planned"
	RecommendationExecuted = "executed"
	RecommendationRejected = "rejected"
)

type Recommendation struct {
	ID                      int64          `json:"id"`
	ResourceID              string         `json:"resource_id"`
	PluginID                string         `json:"plugin_id"`
	AlgorithmID             string         `json:"algorithm_id"`
	ActionType              string         `json:"action_type"`
	Title                   string         `json:"title"`
	Summary                 string         `json:"summary"`
	Current                 map[string]any `json:"current"`
	Proposed                map[string]any `json:"proposed"`
	Parameters              map[string]any `json:"parameters"`
	EstimatedSavingsMonthly float64        `json:"estimated_savings_monthly"`
	Confidence              string         `json:"confidence"`
	Risk                    string         `json:"risk"`
	Evidence                []string       `json:"evidence"`
	PolicyID                string         `json:"policy_id"`
	Status                  string         `json:"status"`
	CreatedAt               time.Time      `json:"created_at"`
	UpdatedAt               time.Time      `json:"updated_at"`
}

type ActionEvent struct {
	VerificationResult *verification.Result `json:"verification_result,omitempty"`
	ID                 int64                `json:"id"`
	RecommendationID   int64                `json:"recommendation_id,omitempty"`
	ResourceID         string               `json:"resource_id"`
	PluginID           string               `json:"plugin_id"`
	ActionType         string               `json:"action_type"`
	Actor              string               `json:"actor"`
	Mode               string               `json:"mode"`
	Result             string               `json:"result"`
	Message            string               `json:"message"`
	Parameters         map[string]any       `json:"parameters"`
	Plan               *plugin.ActionPlan   `json:"plan,omitempty"`
	PluginResult       *plugin.ActionResult `json:"plugin_result,omitempty"`
	PolicyDecision     policy.Decision      `json:"policy_decision"`
	CreatedAt          time.Time            `json:"created_at"`
}

type Store interface {
	Health(ctx context.Context) error
	UpsertResource(ctx context.Context, res resource.Resource) (resource.Resource, error)
	GetResource(ctx context.Context, id string) (resource.Resource, error)
	ListResources(ctx context.Context) ([]resource.Resource, error)
	CreateRecommendation(ctx context.Context, rec Recommendation) (Recommendation, error)
	GetRecommendation(ctx context.Context, id int64) (Recommendation, error)
	ListRecommendations(ctx context.Context) ([]Recommendation, error)
	SetRecommendationStatus(ctx context.Context, id int64, status string) error
	CreateActionEvent(ctx context.Context, event ActionEvent) (ActionEvent, error)
	ListActionEvents(ctx context.Context) ([]ActionEvent, error)
}
