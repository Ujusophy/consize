package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/consize-oss/consize/internal/store"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
)

type fakeDiscovery struct{ observation plugin.ProviderObservation }

func (f fakeDiscovery) ID() string { return "test-discovery" }
func (f fakeDiscovery) Manifest() plugin.Manifest {
	return plugin.Manifest{ID: f.ID(), Category: plugin.CategoryDataSource, SupportedResourceTypes: []string{resource.TypeKubernetesDeployment}, Capabilities: []string{plugin.CapabilityResourceDiscover}}
}
func (f fakeDiscovery) Health(context.Context) plugin.Health { return plugin.Health{Status: "healthy"} }
func (f fakeDiscovery) Discover(context.Context) ([]plugin.ProviderObservation, error) {
	return []plugin.ProviderObservation{f.observation}, nil
}

func TestRunRegistersNormalizedObservation(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	p := fakeDiscovery{observation: plugin.ProviderObservation{
		PluginID: "test-discovery", Provider: resource.ProviderKubernetes, ObservedAt: now,
		Resource: resource.Resource{ID: "k8s:ns:api", Type: resource.TypeKubernetesDeployment, Provider: resource.ProviderKubernetes, ProviderResourceID: "ns/api", Name: "api"},
	}}
	manager := plugin.NewManager()
	if err := manager.RegisterDiscovery(p); err != nil {
		t.Fatal(err)
	}
	registry := store.NewMemory()
	service := New(registry, manager)
	service.now = func() time.Time { return now }
	results, err := service.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Count != 1 {
		t.Fatalf("unexpected results: %#v", results)
	}
	got, err := registry.GetResource(context.Background(), "k8s:ns:api")
	if err != nil {
		t.Fatal(err)
	}
	if got.SourcePluginID != p.ID() || !got.ObservedAt.Equal(now) {
		t.Fatalf("resource was not normalized: %#v", got)
	}
}
