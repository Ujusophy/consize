package cost

import (
	"context"
	"fmt"

	"github.com/consize-oss/consize/internal/store"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
)

// Enrich attaches a transparent price estimate without changing the
// recommendation when pricing is not configured.
func Enrich(ctx context.Context, manager *plugin.Manager, pluginID string, res resource.Resource, rec store.Recommendation) (store.Recommendation, error) {
	if pluginID == "" {
		return rec, nil
	}
	p, err := manager.CostPlugin(pluginID, res.Type)
	if err != nil {
		return rec, fmt.Errorf("cost plugin: %w", err)
	}
	estimate, err := p.Estimate(ctx, res, rec.Proposed)
	if err != nil {
		return rec, fmt.Errorf("estimate recommendation cost: %w", err)
	}
	if estimate.Classification != "estimate" || estimate.ResourceID != res.ID || estimate.Currency == "" || estimate.PricingSource == "" {
		return rec, fmt.Errorf("cost plugin returned incomplete or misclassified evidence")
	}
	rec.CostEstimate = &estimate
	rec.EstimatedSavingsMonthly = estimate.SavingsMonthly
	rec.Evidence = append(rec.Evidence,
		fmt.Sprintf("estimated monthly savings: %.2f %s", estimate.SavingsMonthly, estimate.Currency),
		fmt.Sprintf("pricing source: %s (estimate; not realized billing)", estimate.PricingSource),
	)
	return rec, nil
}
