package marketplace

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/consize-oss/consize/pkg/plugin"
)

func signed(t *testing.T, r Release, expiry time.Time) (Envelope, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(Catalog{Protocol: Protocol, ExpiresAt: expiry, Releases: []Release{r}})
	if err != nil {
		t.Fatal(err)
	}
	return Envelope{Payload: base64.StdEncoding.EncodeToString(b), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, b))}, pub
}
func release(binary []byte) Release {
	hash := sha256.Sum256(binary)
	return Release{Manifest: plugin.Manifest{ID: "test-metrics", Version: "1.0.0", Category: plugin.CategoryMetrics, SupportedResourceTypes: []string{"test.resource"}}, OS: runtime.GOOS, Arch: runtime.GOARCH, URL: "https://releases.example.org/plugin", Size: int64(len(binary)), SHA256: hex.EncodeToString(hash[:])}
}
func TestSignatureExpiryAndTamper(t *testing.T) {
	e, key := signed(t, release([]byte("binary")), time.Now().Add(time.Hour))
	if _, err := Verify(e, key, true); err != nil {
		t.Fatal(err)
	}
	wrong, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := Verify(e, wrong, true); err == nil {
		t.Fatal("wrong publisher accepted")
	}
	e.Payload = base64.StdEncoding.EncodeToString([]byte(`{}`))
	if _, err := Verify(e, key, true); err == nil {
		t.Fatal("tampered catalog accepted")
	}
	e, key = signed(t, release([]byte("binary")), time.Now().Add(-time.Hour))
	if _, err := Verify(e, key, true); err == nil {
		t.Fatal("expired catalog accepted")
	}
	if _, err := Verify(e, key, false); err != nil {
		t.Fatal("pinned recovery must not depend on catalog availability/expiry")
	}
}
func TestRejectUnsafeRelease(t *testing.T) {
	for _, mutate := range []func(*Release){func(r *Release) { r.Manifest.ID = "../escape" }, func(r *Release) { r.URL = "http://insecure.example.org/plugin" }, func(r *Release) { r.SHA256 = "bad" }, func(r *Release) { r.Size = MaxBinarySize + 1 }, func(r *Release) { r.Manifest.CanMutateInfrastructure = true }} {
		r := release([]byte("binary"))
		mutate(&r)
		e, key := signed(t, r, time.Now().Add(time.Hour))
		if _, err := Verify(e, key, true); err == nil {
			t.Fatal("unsafe release accepted")
		}
	}
}
func TestInstalledIntegrity(t *testing.T) {
	binary := []byte("test artifact")
	r := release(binary)
	e, key := signed(t, r, time.Now().Add(time.Hour))
	dir := t.TempDir()
	b, _ := json.Marshal(Receipt{Envelope: e, ID: r.Manifest.ID, Version: r.Manifest.Version, OS: r.OS, Arch: r.Arch})
	if err := os.WriteFile(filepath.Join(dir, "receipt.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin"), binary, 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Installed(dir, key); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte("evil artifact"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Installed(dir, key); err == nil {
		t.Fatal("modified binary accepted")
	}
	os.Remove(filepath.Join(dir, "plugin"))
	os.Symlink("receipt.json", filepath.Join(dir, "plugin"))
	if _, _, err := Installed(dir, key); err == nil {
		t.Fatal("symlink binary accepted")
	}
}
