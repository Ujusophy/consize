package marketplace

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/consize-oss/consize/pkg/plugin"
	"golang.org/x/sys/unix"
)

const Protocol = 1
const MaxBinarySize = 256 << 20

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,100}$`)

// Envelope signs the exact payload bytes, not reserialized JSON.
type Envelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}
type Catalog struct {
	Protocol  int       `json:"protocol"`
	ExpiresAt time.Time `json:"expires_at"`
	Releases  []Release `json:"releases"`
}
type Release struct {
	Manifest plugin.Manifest `json:"manifest"`
	OS       string          `json:"os"`
	Arch     string          `json:"arch"`
	URL      string          `json:"url"`
	SHA256   string          `json:"sha256"`
	Size     int64           `json:"size"`
}
type Receipt struct {
	Envelope Envelope `json:"envelope"`
	ID       string   `json:"id"`
	Version  string   `json:"version"`
	OS       string   `json:"os"`
	Arch     string   `json:"arch"`
}

func PublicKey(path string) (ed25519.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(stringTrim(b))
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, errors.New("public key must be base64 Ed25519")
	}
	return key, nil
}
func stringTrim(b []byte) string {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return string(b)
}

func Verify(envelope Envelope, key ed25519.PublicKey, checkExpiry bool) (Catalog, error) {
	var catalog Catalog
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return catalog, err
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil || len(key) != ed25519.PublicKeySize || !ed25519.Verify(key, payload, signature) {
		return catalog, errors.New("catalog signature rejected")
	}
	if err := json.Unmarshal(payload, &catalog); err != nil {
		return catalog, err
	}
	if catalog.Protocol != Protocol || catalog.ExpiresAt.IsZero() || (checkExpiry && !time.Now().Before(catalog.ExpiresAt)) {
		return catalog, errors.New("catalog protocol incompatible or expired")
	}
	seen := map[string]bool{}
	for _, r := range catalog.Releases {
		digest, err := hex.DecodeString(r.SHA256)
		identity := r.Manifest.ID + "/" + r.Manifest.Version + "/" + r.OS + "/" + r.Arch
		if !identifier.MatchString(r.Manifest.ID) || !identifier.MatchString(r.Manifest.Version) || !identifier.MatchString(r.OS) || !identifier.MatchString(r.Arch) || err != nil || len(digest) != sha256.Size || r.Size <= 0 || r.Size > MaxBinarySize || seen[identity] {
			return catalog, errors.New("invalid or duplicate release")
		}
		if r.Manifest.Category != plugin.CategoryAction && r.Manifest.Category != plugin.CategoryMetrics {
			return catalog, errors.New("unsupported plugin category")
		}
		if len(r.Manifest.SupportedResourceTypes) == 0 {
			return catalog, errors.New("release has no resource types")
		}
		if r.Manifest.Category == plugin.CategoryMetrics && r.Manifest.CanMutateInfrastructure {
			return catalog, errors.New("metrics release cannot declare mutation")
		}
		if err := secureURL(r.URL); err != nil {
			return catalog, err
		}
		seen[identity] = true
	}
	return catalog, nil
}

func secureURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("marketplace URLs must use HTTPS without userinfo or fragments")
	}
	return nil
}
func client() *http.Client {
	return &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return secureURL(req.URL.String())
	}}
}
func get(ctx context.Context, raw string) (*http.Response, error) {
	if err := secureURL(raw); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	return resp, nil
}
func Fetch(ctx context.Context, raw string, key ed25519.PublicKey) (Envelope, Catalog, error) {
	var e Envelope
	resp, err := get(ctx, raw)
	if err != nil {
		return e, Catalog{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, (4<<20)+1))
	if err != nil {
		return e, Catalog{}, err
	}
	if len(b) > 4<<20 {
		return e, Catalog{}, errors.New("catalog too large")
	}
	if err = json.Unmarshal(b, &e); err != nil {
		return e, Catalog{}, err
	}
	c, err := Verify(e, key, true)
	return e, c, err
}

func selectRelease(c Catalog, id, version, osName, arch string) (Release, error) {
	for _, r := range c.Releases {
		if r.Manifest.ID == id && r.Manifest.Version == version && r.OS == osName && r.Arch == arch {
			return r, nil
		}
	}
	return Release{}, errors.New("requested plugin version/platform is not in signed catalog")
}

// Install downloads data only. It never invokes a downloaded program.
func Install(ctx context.Context, root string, e Envelope, key ed25519.PublicKey, id, version string) (string, error) {
	c, err := Verify(e, key, true)
	if err != nil {
		return "", err
	}
	r, err := selectRelease(c, id, version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	// Reject symlinked install roots; installation must be in administrator-owned storage.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return "", errors.New("install root must not contain symlinks")
	}
	lock, err := os.OpenFile(filepath.Join(root, ".install.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return "", errors.New("another plugin installation is active")
	}
	defer unix.Flock(int(lock.Fd()), unix.LOCK_UN)
	destination := filepath.Join(root, id+"-"+version+"-"+runtime.GOOS+"-"+runtime.GOARCH)
	if _, err = os.Lstat(destination); err == nil {
		return "", errors.New("version already installed; refusing overwrite")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	temp, err := os.MkdirTemp(root, ".download-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temp)
	resp, err := get(ctx, r.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	f, err := os.OpenFile(filepath.Join(temp, "plugin"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, hash), io.LimitReader(resp.Body, r.Size+1))
	syncErr := f.Sync()
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if syncErr != nil {
		return "", syncErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n != r.Size || hex.EncodeToString(hash.Sum(nil)) != r.SHA256 {
		return "", errors.New("artifact size or SHA256 rejected")
	}
	receipt := Receipt{Envelope: e, ID: id, Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH}
	b, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(temp, "receipt.json"), b, 0600); err != nil {
		return "", err
	}
	rf, err := os.OpenFile(filepath.Join(temp, "receipt.json"), os.O_RDWR, 0)
	if err != nil {
		return "", err
	}
	err = rf.Sync()
	rf.Close()
	if err != nil {
		return "", err
	}
	if err = os.Chmod(filepath.Join(temp, "plugin"), 0700); err != nil {
		return "", err
	}
	d, err := os.Open(temp)
	if err != nil {
		return "", err
	}
	err = d.Sync()
	d.Close()
	if err != nil {
		return "", err
	}
	if err = os.Rename(temp, destination); err != nil {
		return "", err
	}
	d, err = os.Open(root)
	if err != nil {
		return "", err
	}
	err = d.Sync()
	d.Close()
	if err != nil {
		return "", err
	}
	return destination, nil
}

func Installed(directory string, key ed25519.PublicKey) (Release, string, error) {
	var receipt Receipt
	b, err := os.ReadFile(filepath.Join(directory, "receipt.json"))
	if err != nil {
		return Release{}, "", err
	}
	if len(b) > 4<<20 {
		return Release{}, "", errors.New("receipt too large")
	}
	if err = json.Unmarshal(b, &receipt); err != nil {
		return Release{}, "", err
	}
	// Expiry prevents new installs; pinned installed versions remain usable for recovery.
	c, err := Verify(receipt.Envelope, key, false)
	if err != nil {
		return Release{}, "", err
	}
	if receipt.OS != runtime.GOOS || receipt.Arch != runtime.GOARCH {
		return Release{}, "", errors.New("installed platform mismatch")
	}
	r, err := selectRelease(c, receipt.ID, receipt.Version, receipt.OS, receipt.Arch)
	if err != nil {
		return r, "", err
	}
	path := filepath.Join(directory, "plugin")
	info, err := os.Lstat(path)
	if err != nil {
		return r, "", err
	}
	if !info.Mode().IsRegular() || info.Size() != r.Size {
		return r, "", errors.New("artifact is not the expected regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return r, "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, io.LimitReader(f, r.Size+1)); err != nil {
		return r, "", err
	}
	if hex.EncodeToString(hash.Sum(nil)) != r.SHA256 {
		return r, "", errors.New("installed artifact checksum rejected")
	}
	return r, path, nil
}
