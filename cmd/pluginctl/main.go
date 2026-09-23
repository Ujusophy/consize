// pluginctl manages explicitly trusted executable plugin releases.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/plugin/marketplace"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: pluginctl catalog|install|artifact|sign")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	switch args[0] {
	case "catalog", "install":
		url := fs.String("catalog", "", "HTTPS signed catalog URL")
		keyPath := fs.String("public-key", "", "Trusted publisher public key file (base64 Ed25519)")
		id := fs.String("id", "", "Plugin ID")
		version := fs.String("version", "", "Exact version; no implicit latest")
		root := fs.String("dir", ".consize/plugins", "Install root")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		key, err := marketplace.PublicKey(*keyPath)
		if err != nil {
			return err
		}
		e, c, err := marketplace.Fetch(context.Background(), *url, key)
		if err != nil {
			return err
		}
		if args[0] == "catalog" {
			return json.NewEncoder(os.Stdout).Encode(c)
		}
		if *id == "" || *version == "" {
			return errors.New("install requires id and exact version")
		}
		path, err := marketplace.Install(context.Background(), *root, e, key, *id, *version)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"installed_directory": path, "status": "installed_not_enabled"})
	case "artifact":
		binary := fs.String("binary", "", "Built plugin executable")
		manifest := fs.String("manifest", "", "Plugin manifest JSON file")
		url := fs.String("url", "", "Published HTTPS binary URL")
		osName := fs.String("os", "", "Target OS")
		arch := fs.String("arch", "", "Target architecture")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		var m plugin.Manifest
		b, err := os.ReadFile(*manifest)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, &m); err != nil {
			return err
		}
		f, err := os.Open(*binary)
		if err != nil {
			return err
		}
		defer f.Close()
		hash := sha256.New()
		n, err := io.Copy(hash, f)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(marketplace.Release{Manifest: m, OS: *osName, Arch: *arch, URL: *url, SHA256: hex.EncodeToString(hash.Sum(nil)), Size: n})
	case "sign":
		input := fs.String("input", "", "Catalog JSON file")
		keyPath := fs.String("private-key", "", "Publisher private key file (base64 Ed25519)")
		output := fs.String("out", "", "Signed catalog destination (must not exist)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		payload, err := os.ReadFile(*input)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(*keyPath)
		if err != nil {
			return err
		}
		var keyString string
		for _, c := range b {
			if c != '\n' && c != '\r' {
				keyString += string(c)
			}
		}
		key, err := base64.StdEncoding.DecodeString(keyString)
		if err != nil || len(key) != ed25519.PrivateKeySize {
			return errors.New("invalid Ed25519 private key")
		}
		private := ed25519.PrivateKey(key)
		e := marketplace.Envelope{Payload: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, payload))}
		if _, err = marketplace.Verify(e, private.Public().(ed25519.PublicKey), true); err != nil {
			return err
		}
		f, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		if err = json.NewEncoder(f).Encode(e); err != nil {
			return err
		}
		return f.Sync()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
