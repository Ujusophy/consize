package discovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/consize-oss/consize/internal/store"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
)

type Result struct {
	PluginID  string              `json:"plugin_id"`
	Count     int                 `json:"count"`
	Resources []resource.Resource `json:"resources"`
}

type Service struct {
	store   store.Store
	plugins *plugin.Manager
	now     func() time.Time
}

func New(st store.Store, plugins *plugin.Manager) *Service {
	return &Service{store: st, plugins: plugins, now: time.Now}
}

func (s *Service) Run(ctx context.Context) ([]Result, error) {
	if s.store == nil || s.plugins == nil {
		return nil, errors.New("discovery requires registry and plugin manager")
	}
	plugins := s.plugins.DiscoveryPlugins()
	if len(plugins) == 0 {
		return nil, errors.New("no discovery plugins are configured")
	}
	results := make([]Result, 0, len(plugins))
	for _, provider := range plugins {
		observations, err := provider.Discover(ctx)
		if err != nil {
			return nil, fmt.Errorf("discover with %s: %w", provider.ID(), err)
		}
		result := Result{PluginID: provider.ID(), Resources: make([]resource.Resource, 0, len(observations))}
		for _, observation := range observations {
			res, err := normalize(provider, observation, s.now().UTC())
			if err != nil {
				return nil, err
			}
			persisted, err := s.store.UpsertResource(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("register discovered resource %s: %w", res.ID, err)
			}
			result.Resources = append(result.Resources, persisted)
		}
		result.Count = len(result.Resources)
		results = append(results, result)
	}
	return results, nil
}

func normalize(provider plugin.DiscoveryPlugin, observation plugin.ProviderObservation, now time.Time) (resource.Resource, error) {
	manifest := provider.Manifest()
	res := observation.Resource
	if observation.PluginID != provider.ID() || observation.Provider == "" || res.Provider != observation.Provider {
		return resource.Resource{}, fmt.Errorf("discovery plugin %s returned mismatched provider identity", provider.ID())
	}
	if res.ID == "" || res.Type == "" || res.ProviderResourceID == "" {
		return resource.Resource{}, fmt.Errorf("discovery plugin %s returned incomplete resource identity", provider.ID())
	}
	if !plugin.Supports(manifest.SupportedResourceTypes, res.Type) {
		return resource.Resource{}, fmt.Errorf("discovery plugin %s returned unsupported resource type %s", provider.ID(), res.Type)
	}
	if observation.ObservedAt.IsZero() || observation.ObservedAt.After(now.Add(time.Minute)) {
		return resource.Resource{}, fmt.Errorf("discovery plugin %s returned invalid observation time", provider.ID())
	}
	res.SourcePluginID = provider.ID()
	res.ObservedAt = observation.ObservedAt.UTC()
	return res, nil
}
