package repositorysecurity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestContentRulesRedactCredentialValues(t *testing.T) {
	file := TrackedFile{Path: "config/runtime.env", ContentSet: true, Content: []byte("API_" + "TOKEN" + "=" + "real-looking-value-12345\n")}
	findings := scanContent(file)
	if !hasRule(findings, "embedded-credential") {
		t.Fatalf("expected embedded-credential finding: %#v", findings)
	}
	for _, finding := range findings {
		if strings.Contains(finding.Summary, "real-looking-value") {
			t.Fatal("finding summary exposed credential value")
		}
	}
}

func TestPlaceholderExampleDoesNotTriggerEmbeddedCredential(t *testing.T) {
	file := TrackedFile{Path: ".env.example", ContentSet: true, Content: []byte("API_" + "TOKEN" + "=" + "change-me\n")}
	findings := scanContent(file)
	if hasRule(findings, "embedded-credential") {
		t.Fatalf("placeholder was treated as a credential: %#v", findings)
	}
}

func TestFrontendServerConfigurationExposure(t *testing.T) {
	file := TrackedFile{Path: "ui/app/page.tsx", ContentSet: true, Content: []byte("\"use client\"; const key = process.env.DATABASE_URL")}
	if findings := scanContent(file); !hasRule(findings, "server-config-in-client") {
		t.Fatalf("expected server-config-in-client finding: %#v", findings)
	}
}

func TestWorkflowRules(t *testing.T) {
	unsafe := TrackedFile{Path: ".github/workflows/unsafe.yml", ContentSet: true, Content: []byte(`
on: pull_request_target
jobs:
  test:
    permissions: write-all
    steps:
      - uses: actions/checkout@v4
      - run: echo "${{ github.event.pull_request.title }}"
`)}
	findings := scanWorkflow(unsafe)
	if !hasRule(findings, "pull-request-target") {
		t.Fatalf("expected pull-request-target finding: %#v", findings)
	}

	safe := TrackedFile{Path: ".github/workflows/safe.yml", ContentSet: true, Content: []byte(`
on: [push, pull_request]
permissions:
  contents: read
jobs:
  test:
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
      - run: go test ./...
`)}
	if findings := scanWorkflow(safe); len(findings) != 0 {
		t.Fatalf("safe workflow produced findings: %#v", findings)
	}
}

func TestDeclarationsRequireExactDigestAndExpire(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	registry := DeclarationRegistry{Schema: "./repository-allowlist.schema.json", SchemaVersion: 1, Declarations: []Declaration{{
		ID: "REP-ALLOW-2026-0001", RuleID: "embedded-credential", Path: "fixtures/token.txt",
		SHA256: digest([]byte("fixture")), Classification: "test_fixture",
		Reason: "A synthetic scanner fixture with no live access.", Owner: "github:owner", ApprovedBy: "github:reviewer",
		ReviewedAt: now.Add(-time.Hour).Format(time.RFC3339), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
	}}}
	if err := ValidateDeclarations(registry, now); err != nil {
		t.Fatal(err)
	}
	registry.Declarations[0].ExpiresAt = now.Add(-time.Minute).Format(time.RFC3339)
	if err := ValidateDeclarations(registry, now); err == nil {
		t.Fatal("expected expired declaration to fail")
	}
}

func TestIgnoreRulesCoverRequiredLocalState(t *testing.T) {
	content := []byte(strings.Join([]string{
		".env", ".env.*", ".consize/", ".gocache/", "*.jsonl", "*.log", "ui/node_modules/", "ui/.next/", "ui/coverage/", "coverage/", ".tools/", ".cache/security/", "tmp/", "output/", "bin/", "dist/", "*.test", "site/",
	}, "\n"))
	files := map[string]TrackedFile{".gitignore": {Path: ".gitignore", Content: content, ContentSet: true}}
	if findings := scanIgnoreFiles(files); len(findings) != 0 {
		t.Fatalf("complete ignore file produced findings: %#v", findings)
	}
}

func TestDeclarationLoaderRejectsUnknownFields(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "allowlist.json")
	if err := os.WriteFile(path, []byte(`{"$schema":"./repository-allowlist.schema.json","schema_version":1,"declarations":[],"extra":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDeclarations(path, time.Now()); err == nil {
		t.Fatal("expected unknown field to fail")
	}
}

func hasRule(findings []Candidate, rule string) bool {
	for _, finding := range findings {
		if finding.RuleID == rule {
			return true
		}
	}
	return false
}
