package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/consize-oss/consize/pkg/resource"
)

func TestResourceUpsertPreservesIdentityAndFirstSeen(t *testing.T) {
	ctx := context.Background()
	st := NewMemory()
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	st.now = func() time.Time { return now }

	created, err := st.UpsertResource(ctx, resourceInput(now))
	if err != nil {
		t.Fatal(err)
	}
	st.now = func() time.Time { return now.Add(time.Hour) }
	update := resourceInput(now.Add(time.Hour))
	update.Owner = "payments-platform"
	update.CurrentState["memory_request_bytes"] = float64(384 * 1024 * 1024)
	updated, err := st.UpsertResource(ctx, update)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != created.ID || !updated.FirstSeenAt.Equal(created.FirstSeenAt) || !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("stable fields changed: created=%#v updated=%#v", created, updated)
	}
	if updated.Owner != "payments-platform" {
		t.Fatalf("mutable ownership was not updated: %#v", updated)
	}
}

func TestResourceUpsertRejectsIdentityMutationAndCollision(t *testing.T) {
	ctx := context.Background()
	st := NewMemory()
	now := time.Now().UTC()
	first, err := st.UpsertResource(ctx, resourceInput(now))
	if err != nil {
		t.Fatal(err)
	}

	mutated := resourceInput(now.Add(time.Minute))
	mutated.ID = first.ID
	mutated.Account = "another-cluster"
	if _, err := st.UpsertResource(ctx, mutated); !errors.Is(err, resource.ErrIdentityConflict) {
		t.Fatalf("identity mutation error = %v", err)
	}

	duplicate := resourceInput(now.Add(time.Minute))
	duplicate.ID = "another-legacy-id"
	if _, err := st.UpsertResource(ctx, duplicate); !errors.Is(err, resource.ErrIdentityConflict) {
		t.Fatalf("collision error = %v", err)
	}
}

func TestResourceUpsertKeepsLegacyIDDuringCanonicalRediscovery(t *testing.T) {
	ctx := context.Background()
	st := NewMemory()
	now := time.Now().UTC()
	legacy := resourceInput(now)
	legacy.ID = "k8s:payments:checkout-api"
	created, err := st.UpsertResource(ctx, legacy)
	if err != nil {
		t.Fatal(err)
	}
	rediscovered, err := st.UpsertResource(ctx, resourceInput(now.Add(time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	if rediscovered.ID != created.ID || resource.IsCanonicalID(rediscovered.ID) {
		t.Fatalf("legacy references were not preserved: %#v", rediscovered)
	}
}

func TestResourceUpsertRejectsStaleObservationAndInvalidLifecycle(t *testing.T) {
	ctx := context.Background()
	st := NewMemory()
	now := time.Now().UTC()
	if _, err := st.UpsertResource(ctx, resourceInput(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertResource(ctx, resourceInput(now.Add(-time.Minute))); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale observation error = %v", err)
	}
	deleted := resourceInput(now.Add(time.Minute))
	deleted.LifecycleState = resource.LifecycleDeleted
	if _, err := st.UpsertResource(ctx, deleted); err != nil {
		t.Fatal(err)
	}
	stale := resourceInput(now.Add(2 * time.Minute))
	stale.LifecycleState = resource.LifecycleStale
	if _, err := st.UpsertResource(ctx, stale); err == nil || !strings.Contains(err.Error(), "lifecycle") {
		t.Fatalf("lifecycle error = %v", err)
	}
}

func resourceInput(observedAt time.Time) resource.Resource {
	return resource.Resource{
		Type: resource.TypeKubernetesDeployment, Provider: resource.ProviderKubernetes,
		ProviderResourceID: "payments/checkout-api", Name: "checkout-api",
		Environment: resource.EnvProduction, Owner: "payments", Region: "us-east-1",
		Account: "cluster-a", Criticality: resource.CriticalityHigh,
		ObservedAt: observedAt, Metadata: map[string]any{"namespace": "payments"},
		CurrentState: map[string]any{"memory_request_bytes": float64(512 * 1024 * 1024)},
	}
}
