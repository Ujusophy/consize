package gitleaksreport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/consize-oss/consize/internal/securitypolicy"
)

type rawFinding struct {
	RuleID      string   `json:"RuleID"`
	Description string   `json:"Description"`
	File        string   `json:"File"`
	SymlinkFile string   `json:"SymlinkFile"`
	Commit      string   `json:"Commit"`
	StartLine   int      `json:"StartLine"`
	EndLine     int      `json:"EndLine"`
	StartColumn int      `json:"StartColumn"`
	EndColumn   int      `json:"EndColumn"`
	Secret      string   `json:"Secret"`
	Match       string   `json:"Match"`
	Fingerprint string   `json:"Fingerprint"`
	Entropy     float64  `json:"Entropy"`
	Author      string   `json:"Author"`
	Email       string   `json:"Email"`
	Date        string   `json:"Date"`
	Message     string   `json:"Message"`
	Tags        []string `json:"Tags"`
	Link        string   `json:"Link"`
}

type Options struct {
	Context     string
	Version     string
	ScanID      string
	Now         time.Time
	StripPrefix string
}

func Normalize(input io.Reader, options Options) (securitypolicy.FindingReport, error) {
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	var raw []rawFinding
	if err := decoder.Decode(&raw); err != nil {
		return securitypolicy.FindingReport{}, fmt.Errorf("decode Gitleaks report: %w", err)
	}
	report := securitypolicy.FindingReport{
		Schema: "./findings.schema.json", SchemaVersion: securitypolicy.SchemaVersion,
		Scanner: "gitleaks", ScanID: options.ScanID, GeneratedAt: options.Now.UTC().Format(time.RFC3339),
		Status: "completed", Attempts: 1,
		Target:   securitypolicy.ReportTarget{Component: "consize-repository", Environment: "ci", Context: options.Context, Version: options.Version},
		Findings: []securitypolicy.Finding{},
	}
	for _, item := range raw {
		if item.RuleID == "" || item.File == "" {
			return securitypolicy.FindingReport{}, fmt.Errorf("Gitleaks finding is missing rule or path")
		}
		if item.Secret != "" && item.Secret != "REDACTED" || strings.Contains(item.Match, "CONSIZE_TEST_TOKEN_") {
			return securitypolicy.FindingReport{}, fmt.Errorf("Gitleaks report contains unredacted matched material")
		}
		path := filepath.ToSlash(strings.TrimPrefix(item.File, options.StripPrefix))
		path = strings.TrimPrefix(path, "./")
		identity := fmt.Sprintf("%s|%s|%s|%d", item.RuleID, item.Commit, path, item.StartLine)
		digest := sha256.Sum256([]byte(identity))
		report.Findings = append(report.Findings, securitypolicy.Finding{
			Fingerprint: "sha256:" + hex.EncodeToString(digest[:]), RuleID: item.RuleID,
			FindingType: "secret", Severity: "critical", Confidence: "high", Reachability: "not_applicable",
			Mandatory: true, ExceptionAllowed: true, LiveCredential: false,
			Component: "consize-repository", ComponentType: "source", Path: path,
			Environment: "ci", Context: options.Context, Version: options.Version,
			Summary:  fmt.Sprintf("Potential secret detected by rule %s at %s:%d (commit %s); value redacted", item.RuleID, path, item.StartLine, shortCommit(item.Commit)),
			Redacted: true, Owner: "security",
		})
	}
	return report, nil
}

func shortCommit(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	if value == "" {
		return "working-tree"
	}
	return value
}
