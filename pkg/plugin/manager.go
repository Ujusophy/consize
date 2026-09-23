package plugin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	ErrNotFound          = errors.New("plugin not found")
	ErrCapabilityMissing = errors.New("plugin capability missing")
	ErrUnsupported       = errors.New("plugin does not support requested resource or action")
)

type Manager struct {
	mu        sync.RWMutex
	actions   map[string]ActionPlugin
	metrics   map[string]MetricsPlugin
	discovery map[string]DiscoveryPlugin
}

func NewManager() *Manager {
	return &Manager{actions: map[string]ActionPlugin{}, metrics: map[string]MetricsPlugin{}, discovery: map[string]DiscoveryPlugin{}}
}

func (m *Manager) RegisterDiscovery(p DiscoveryPlugin) error {
	if p == nil {
		return errors.New("plugin is nil")
	}
	manifest := p.Manifest()
	if manifest.ID == "" || !Supports(manifest.Capabilities, CapabilityResourceDiscover) {
		return errors.New("discovery plugin must declare an id and resource.discover capability")
	}
	if manifest.CanMutateInfrastructure && !Supports(manifest.Capabilities, CapabilityActionExecute) {
		return fmt.Errorf("discovery-only plugin %s cannot mutate infrastructure", manifest.ID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.discovery[manifest.ID]; exists {
		return fmt.Errorf("discovery plugin %s already registered", manifest.ID)
	}
	m.discovery[manifest.ID] = p
	return nil
}

func (m *Manager) DiscoveryPlugins() []DiscoveryPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]DiscoveryPlugin, 0, len(m.discovery))
	for _, p := range m.discovery {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

func (m *Manager) RegisterAction(p ActionPlugin) error {
	if p == nil {
		return errors.New("plugin is nil")
	}
	manifest := p.Manifest()
	if manifest.ID == "" {
		return errors.New("plugin manifest id is required")
	}
	if manifest.Category != CategoryAction {
		return fmt.Errorf("plugin %s category must be %q", manifest.ID, CategoryAction)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.actions[manifest.ID]; exists {
		return fmt.Errorf("plugin %s already registered", manifest.ID)
	}
	if _, exists := m.metrics[manifest.ID]; exists {
		return fmt.Errorf("plugin %s already registered", manifest.ID)
	}
	m.actions[manifest.ID] = p
	return nil
}

func (m *Manager) RegisterMetrics(p MetricsPlugin) error {
	if p == nil {
		return errors.New("plugin is nil")
	}
	manifest := p.Manifest()
	if manifest.ID == "" {
		return errors.New("plugin manifest id is required")
	}
	if manifest.Category != CategoryMetrics && manifest.Category != CategoryCost && manifest.Category != CategoryDataSource {
		return fmt.Errorf("plugin %s category must be read-oriented", manifest.ID)
	}
	if manifest.CanMutateInfrastructure {
		return fmt.Errorf("metrics plugin %s cannot mutate infrastructure", manifest.ID)
	}
	if !Supports(manifest.Capabilities, CapabilityMetricsRead) && !Supports(manifest.Capabilities, CapabilityCostRead) {
		return fmt.Errorf("metrics plugin %s must expose a read capability", manifest.ID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.actions[manifest.ID]; exists {
		return fmt.Errorf("plugin %s already registered", manifest.ID)
	}
	if _, exists := m.metrics[manifest.ID]; exists {
		return fmt.Errorf("plugin %s already registered", manifest.ID)
	}
	m.metrics[manifest.ID] = p
	return nil
}

func (m *Manager) ActionPlugin(id, resourceType, actionType string) (ActionPlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.actions[id]
	if !ok {
		return nil, ErrNotFound
	}
	manifest := p.Manifest()
	if !Supports(manifest.SupportedResourceTypes, resourceType) || !Supports(manifest.SupportedActionTypes, actionType) {
		return nil, ErrUnsupported
	}
	if !Supports(manifest.Capabilities, CapabilityActionPlan) {
		return nil, ErrCapabilityMissing
	}
	return p, nil
}

func (m *Manager) MetricsPlugin(id, resourceType string) (MetricsPlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.metrics[id]
	if !ok {
		return nil, ErrNotFound
	}
	if !Supports(p.Manifest().SupportedResourceTypes, resourceType) {
		return nil, ErrUnsupported
	}
	return p, nil
}

func (m *Manager) MetricsPlugins(resourceType string) []MetricsPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]MetricsPlugin, 0, len(m.metrics))
	for _, p := range m.metrics {
		if Supports(p.Manifest().SupportedResourceTypes, resourceType) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

func (m *Manager) Manifests() []Manifest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	byID := make(map[string]Manifest, len(m.actions)+len(m.metrics)+len(m.discovery))
	for _, p := range m.actions {
		byID[p.ID()] = p.Manifest()
	}
	for _, p := range m.metrics {
		byID[p.ID()] = p.Manifest()
	}
	for _, p := range m.discovery {
		byID[p.ID()] = p.Manifest()
	}
	out := make([]Manifest, 0, len(byID))
	for _, manifest := range byID {
		out = append(out, manifest)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Manager) Health(ctx context.Context, id string) (Health, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.actions[id]; ok {
		return p.Health(ctx), nil
	}
	if p, ok := m.metrics[id]; ok {
		return p.Health(ctx), nil
	}
	if p, ok := m.discovery[id]; ok {
		return p.Health(ctx), nil
	}
	return Health{}, ErrNotFound
}
