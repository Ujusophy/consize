package cibaseline

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFormattingGateRejectsWithoutRepairing(t *testing.T) {
	root := t.TempDir()
	if b, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("init: %s %v", b, err)
	}
	file := filepath.Join(root, "fixture.go")
	bad := []byte("package fixture\nfunc  Bad( ){}\n")
	if err := os.WriteFile(file, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if b, err := exec.Command("git", "-C", root, "add", "fixture.go").CombinedOutput(); err != nil {
		t.Fatalf("add: %s %v", b, err)
	}
	script, err := filepath.Abs("../../scripts/ci/check-go-format.sh")
	if err != nil {
		t.Fatal(err)
	}
	if b, err := exec.Command("bash", script, root).CombinedOutput(); err == nil || !bytes.Contains(b, []byte("Formatting failure")) {
		t.Fatalf("bad formatting did not fail: %s %v", b, err)
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(after, bad) {
		t.Fatal("formatting gate repaired source")
	}
	if b, err := exec.Command("gofmt", "-w", file).CombinedOutput(); err != nil {
		t.Fatalf("fixture setup: %s %v", b, err)
	}
	if b, err := exec.Command("bash", script, root).CombinedOutput(); err != nil {
		t.Fatalf("clean fixture failed: %s %v", b, err)
	}
}
