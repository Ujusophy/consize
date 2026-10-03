package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/consize-oss/consize/pkg/resource"
)

func completeResource(id string) resource.Resource {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return resource.Resource{
		SchemaVersion: resource.CurrentSchemaVersion,
		ID:            id, Type: resource.TypeKubernetesDeployment,
		Provider: resource.ProviderKubernetes, ProviderResourceID: "default/app",
		Name: "app", Environment: resource.EnvDevelopment, Owner: "platform",
		Region: "local", Account: "test-cluster", Criticality: resource.CriticalityLow,
		LifecycleState: resource.LifecycleActive, SupportStatus: resource.SupportFull,
		Labels: map[string]string{}, Metadata: map[string]any{}, CurrentState: map[string]any{"size": 10},
		ObservedAt: now, FirstSeenAt: now, LastSeenAt: now, CreatedAt: now, UpdatedAt: now,
	}
}

func TestDurableStoreMigratesV1State(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	legacyResource := completeResource("one")
	legacyResource.SchemaVersion = 0
	legacyResource.LifecycleState = ""
	legacyResource.SupportStatus = ""
	legacyResource.ObservedAt = time.Time{}
	legacyResource.FirstSeenAt = time.Time{}
	legacyResource.LastSeenAt = time.Time{}
	legacy := diskState{
		Version:         1,
		Resources:       map[string]resource.Resource{"one": legacyResource},
		Recommendations: map[int64]Recommendation{}, Actions: map[int64]ActionEvent{}, Jobs: map[int64]Job{},
		NextActionID: 1, NextRecID: 1,
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	st, err := OpenDurable(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var migrated diskState
	if err := json.Unmarshal(data, &migrated); err != nil {
		t.Fatal(err)
	}
	if migrated.Version != currentStateVersion {
		t.Fatalf("version = %d", migrated.Version)
	}
	got := migrated.Resources["one"]
	if got.SchemaVersion != resource.CurrentSchemaVersion || got.LifecycleState != resource.LifecycleActive || got.SupportStatus != resource.SupportFull || got.ObservedAt.IsZero() || got.FirstSeenAt.IsZero() || got.LastSeenAt.IsZero() {
		t.Fatalf("resource was not migrated: %#v", got)
	}
}

func TestDurableStorePersistsAndLocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, err := OpenDurable(path)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := OpenDurable(path); err == nil {
		other.Close()
		t.Fatal("two owners acquired state")
	}
	input := completeResource("one")
	if _, err := st.UpsertResource(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	input.CurrentState["size"] = 999
	st.Close()
	st, err = OpenDurable(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	got, err := st.GetResource(context.Background(), "one")
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentState["size"].(interface{ String() string }).String() != "10" {
		t.Fatal("state was aliased or lost")
	}
}

func TestInvalidStateFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if st, err := OpenDurable(path); err == nil {
		st.Close()
		t.Fatal("corrupt store was reset")
	}
}

func TestPersistenceFailurePoisonsStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	st, err := OpenDurable(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.path = filepath.Join(dir, "missing", "state.json")
	if _, err := st.UpsertResource(context.Background(), completeResource("one")); err == nil {
		t.Fatal("write failure ignored")
	}
	if st.Health(context.Background()) == nil {
		t.Fatal("failed persistence did not block execution")
	}
}
