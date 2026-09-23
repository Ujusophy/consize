package prometheus

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/consize-oss/consize/pkg/resource"
)

type fakeClient struct {
	queries []string
}

type unusableClient struct{}

func (unusableClient) QueryRange(_ context.Context, _ string, start, end time.Time, _ time.Duration) ([]Series, error) {
	return []Series{{Points: []Point{{Timestamp: start.Add(-time.Minute), Value: 100}, {Timestamp: end.Add(time.Minute), Value: 100}, {Timestamp: end, Value: math.NaN()}}}}, nil
}
func TestUnavailableEvidenceHasNoNumericSummary(t *testing.T) {
	p := NewWithClient(unusableClient{}, Config{Window: time.Hour, Step: time.Minute})
	s, err := p.ReadMetrics(context.Background(), resource.Resource{ID: "test", Type: resource.TypeKubernetesDeployment, Name: "test", Metadata: map[string]any{"namespace": "test"}})
	if err != nil {
		t.Fatal(err)
	}
	for key := range defaultQueries() {
		if _, ok := s.Signals[key+"_max"]; ok {
			t.Fatalf("missing %s presented as measured zero", key)
		}
		if s.Coverage[key].ActualPoints != 0 || s.Signals[key+"_points"] != 0 {
			t.Fatal("invalid/out-of-window evidence counted")
		}
	}
}

func (f *fakeClient) QueryRange(_ context.Context, query string, start, end time.Time, step time.Duration) ([]Series, error) {
	f.queries = append(f.queries, query)
	return []Series{{
		Metric: map[string]string{},
		Points: []Point{
			{Timestamp: start, Value: 1},
			{Timestamp: start.Add(step), Value: 2},
			{Timestamp: end, Value: 3},
		},
	}}, nil
}

func TestReadMetricsQueriesPrometheusForDeploymentSignals(t *testing.T) {
	client := &fakeClient{}
	p := NewWithClient(client, Config{Window: time.Hour, Step: time.Minute})
	snapshot, err := p.ReadMetrics(context.Background(), resource.Resource{
		ID:       "k8s:prod:checkout-api",
		Type:     resource.TypeKubernetesDeployment,
		Name:     "checkout-api",
		Metadata: map[string]any{"namespace": "prod", "name": "checkout-api"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.PluginID != ID {
		t.Fatalf("plugin id = %q", snapshot.PluginID)
	}
	if got := snapshot.Signals["cpu_cores_p95"]; got != float64(3) {
		t.Fatalf("cpu p95 = %#v", got)
	}
	if len(client.queries) != 6 {
		t.Fatalf("queries = %d, want 6", len(client.queries))
	}
	for _, key := range []string{"cpu_throttling_ratio_p95", "oom_events_30m_max", "evicted_pods_max", "cpu_cores_p99"} {
		if _, ok := snapshot.Signals[key]; !ok {
			t.Fatalf("missing health signal %s", key)
		}
	}
	for _, query := range client.queries {
		if !strings.Contains(query, `namespace="prod"`) {
			t.Fatalf("query missing namespace matcher: %s", query)
		}
	}
}
