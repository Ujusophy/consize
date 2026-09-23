package orchestrator_test

import (
	"context"
	"testing"

	"github.com/consize-oss/consize/internal/orchestrator"
	"github.com/consize-oss/consize/internal/policy"
	"github.com/consize-oss/consize/internal/store"
	"github.com/consize-oss/consize/pkg/plugin"
	k8splugin "github.com/consize-oss/consize/pkg/plugins/kubernetes"
	"github.com/consize-oss/consize/pkg/resource"
)

type fakePatcher struct {
	patches int
	req     int64
	lim     int64
}

func (f *fakePatcher) Health(context.Context) error { return nil }

func (f *fakePatcher) ReadDeploymentResources(context.Context, string, string, string) (int64, int64, error) {
	return f.req, f.lim, nil
}

func (f *fakePatcher) PatchDeployment(context.Context, string, string, k8splugin.PatchDiff) error {
	f.patches++
	return nil
}

func TestOrchestratorPlansDryRunWithoutExecuting(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemory()
	plugins := plugin.NewManager()
	patcher := &fakePatcher{req: 1024, lim: 2048}
	if err := plugins.RegisterAction(k8splugin.NewWithPatcher(patcher)); err != nil {
		t.Fatal(err)
	}
	res, rec := testResourceAndRecommendation()
	if _, err := st.UpsertResource(ctx, res); err != nil {
		t.Fatal(err)
	}
	rec, err := st.CreateRecommendation(ctx, rec)
	if err != nil {
		t.Fatal(err)
	}
	out, err := orchestrator.New(st, plugins, policy.NewEngine()).ExecuteRecommendation(ctx, rec.ID, "dry_run", "operator@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if out.Executed != nil {
		t.Fatal("dry run should not execute")
	}
	if patcher.patches != 0 {
		t.Fatalf("patches = %d, want 0", patcher.patches)
	}
	events, err := st.ListActionEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("audit events = %d, want 2", len(events))
	}
}

func TestLegacyOrchestratorCannotBypassDurableSafety(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemory()
	plugins := plugin.NewManager()
	patcher := &fakePatcher{req: 1024, lim: 2048}
	if err := plugins.RegisterAction(k8splugin.NewWithPatcher(patcher)); err != nil {
		t.Fatal(err)
	}
	res, rec := testResourceAndRecommendation()
	if _, err := st.UpsertResource(ctx, res); err != nil {
		t.Fatal(err)
	}
	rec, err := st.CreateRecommendation(ctx, rec)
	if err != nil {
		t.Fatal(err)
	}
	out, err := orchestrator.New(st, plugins, policy.NewEngine()).ExecuteRecommendation(ctx, rec.ID, "approved", "operator@example.com")
	if err == nil || out.Executed != nil || patcher.patches != 0 {
		t.Fatal("legacy executor bypassed durable safety")
	}
}

func testResourceAndRecommendation() (resource.Resource, store.Recommendation) {
	res := resource.Resource{
		ID:          "k8s:prod:checkout-api",
		Type:        resource.TypeKubernetesDeployment,
		Provider:    resource.ProviderKubernetes,
		Name:        "checkout-api",
		Environment: resource.EnvProduction,
		Owner:       "payments-team",
		Criticality: resource.CriticalityHigh,
		Metadata:    map[string]any{"namespace": "prod", "name": "checkout-api"},
	}
	rec := store.Recommendation{
		ResourceID: "k8s:prod:checkout-api",
		PluginID:   "kubernetes-action",
		ActionType: "k8s.patch_resources",
		Title:      "Reduce memory request",
		Parameters: map[string]any{
			"patch": map[string]any{
				"resource":         "memory",
				"current_request":  int64(1024),
				"proposed_request": int64(768),
				"current_limit":    int64(2048),
				"proposed_limit":   int64(2048),
			},
		},
	}
	return res, rec
}
