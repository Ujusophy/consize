package store

import (
	"context"
	"github.com/consize-oss/consize/pkg/resource"
	"os"
	"path/filepath"
	"testing"
)

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
	input := resource.Resource{ID: "one", Type: "test", CurrentState: map[string]any{"size": 10}}
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
	if _, err := st.UpsertResource(context.Background(), resource.Resource{ID: "one", Type: "test"}); err == nil {
		t.Fatal("write failure ignored")
	}
	if st.Health(context.Background()) == nil {
		t.Fatal("failed persistence did not block execution")
	}
}
