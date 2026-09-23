package pricing

import (
	"context"
	"testing"

	"github.com/consize-oss/consize/pkg/resource"
)

func TestEstimateIsExplicitlyNotRealizedBilling(t *testing.T) {
	p, err := New(Config{Currency: "USD", Source: "test-rate-card", MemoryGiBMonthly: 10})
	if err != nil {
		t.Fatal(err)
	}
	estimate, err := p.Estimate(context.Background(), resource.Resource{
		ID: "k8s:test:app", Type: resource.TypeKubernetesDeployment,
		CurrentState: map[string]any{"memory_request_bytes": int64(1024 * 1024 * 1024)},
	}, map[string]any{"resource": "memory", "request": int64(512 * 1024 * 1024)})
	if err != nil {
		t.Fatal(err)
	}
	if estimate.Classification != "estimate" || estimate.CurrentMonthly != 10 || estimate.ProposedMonthly != 5 || estimate.SavingsMonthly != 5 {
		t.Fatalf("unexpected estimate: %+v", estimate)
	}
	if realized, _ := estimate.Evidence["billing_realized"].(bool); realized {
		t.Fatal("rate-card estimate was represented as realized billing")
	}
}
