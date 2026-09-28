package repositorysecurity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/consize-oss/consize/internal/securitypolicy"
)

func Scan(ctx context.Context, root string, declarations DeclarationRegistry, options ScanOptions) (securitypolicy.FindingReport, Result, error) {
	files, revision, err := LoadGitIndex(ctx, root)
	if err != nil {
		return securitypolicy.FindingReport{}, Result{}, err
	}
	if options.Revision == "" {
		options.Revision = revision
	}
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	if options.MaxBytes <= 0 {
		options.MaxBytes = DefaultMaxBytes
	}
	byPath := make(map[string]TrackedFile, len(files))
	var candidates []Candidate
	for _, file := range files {
		byPath[file.Path] = file
		if generatedOrLocalPath(file.Path) {
			candidates = append(candidates, configurationCandidate("tracked-local-or-generated-file", file.Path, "local runtime or generated output is tracked", "high"))
		}
		if file.Size > options.MaxBytes {
			candidates = append(candidates, configurationCandidate("oversized-generated-file", file.Path, "tracked file exceeds the repository size limit", "high"))
		}
		if file.Mode == "100755" && !scriptFile(file) {
			candidates = append(candidates, configurationCandidate("unexpected-executable-mode", file.Path, "tracked non-script file has executable mode", "high"))
		}
		if file.Mode == "120000" || file.Mode == "160000" {
			candidates = append(candidates, configurationCandidate("unsupported-git-entry", file.Path, "symlinks and submodules require explicit repository review", "high"))
		}
		candidates = append(candidates, scanContent(file)...)
		candidates = append(candidates, scanWorkflow(file)...)
	}
	candidates = append(candidates, scanIgnoreFiles(byPath)...)
	candidates = uniqueCandidates(candidates)

	result := applyDeclarations(candidates, files, declarations)
	for _, declaration := range result.UnusedDeclarations {
		result.Candidates = append(result.Candidates, configurationCandidate("stale-repository-declaration", ".github/security/repository-allowlist.json", "repository declaration no longer matches an active finding: "+declaration.ID, "high"))
	}
	result.Candidates = uniqueCandidates(result.Candidates)
	report := makeReport(options, result.Candidates)
	return report, result, nil
}

func scriptFile(file TrackedFile) bool {
	return file.ContentSet && len(file.Content) >= 2 && string(file.Content[:2]) == "#!"
}

func applyDeclarations(candidates []Candidate, files []TrackedFile, registry DeclarationRegistry) Result {
	digests := map[string]string{}
	for _, file := range files {
		digests[file.Path] = file.SHA256
	}
	used := map[string]bool{}
	result := Result{}
	for _, candidate := range candidates {
		matched := ""
		for _, declaration := range registry.Declarations {
			if declaration.RuleID == candidate.RuleID && declaration.Path == candidate.Path && declaration.SHA256 == digests[candidate.Path] {
				matched = declaration.ID
				used[declaration.ID] = true
				break
			}
		}
		if matched == "" {
			result.Candidates = append(result.Candidates, candidate)
		} else {
			result.Suppressed = append(result.Suppressed, SuppressedFinding{Candidate: candidate, DeclarationID: matched})
		}
	}
	for _, declaration := range registry.Declarations {
		if !used[declaration.ID] {
			result.UnusedDeclarations = append(result.UnusedDeclarations, declaration)
		}
	}
	return result
}

func makeReport(options ScanOptions, candidates []Candidate) securitypolicy.FindingReport {
	findings := make([]securitypolicy.Finding, 0, len(candidates))
	for _, candidate := range candidates {
		fingerprint := sha256.Sum256([]byte(candidate.RuleID + "\x00" + candidate.Path))
		findings = append(findings, securitypolicy.Finding{
			Fingerprint: "sha256:" + hex.EncodeToString(fingerprint[:]), RuleID: candidate.RuleID,
			FindingType: candidate.FindingType, Severity: candidate.Severity, Confidence: "high",
			Reachability: "not_applicable", Mandatory: true, ExceptionAllowed: candidate.ExceptionAllowed,
			Component: "repository", ComponentType: candidate.ComponentType, Path: candidate.Path,
			Environment: options.Environment, Context: options.Context, Version: options.Revision,
			Summary: candidate.Summary, Redacted: true,
		})
	}
	scan := sha256.Sum256([]byte(options.Revision + options.GeneratedAt.UTC().Format(time.RFC3339Nano)))
	return securitypolicy.FindingReport{
		Schema: "./findings.schema.json", SchemaVersion: securitypolicy.SchemaVersion, Scanner: ScannerID,
		ScanID: fmt.Sprintf("repo-%x", scan[:8]), GeneratedAt: options.GeneratedAt.UTC().Format(time.RFC3339), Status: "completed", Attempts: 1,
		Target:   securitypolicy.ReportTarget{Component: "repository", Environment: options.Environment, Context: options.Context, Version: options.Revision},
		Findings: findings,
	}
}

func uniqueCandidates(candidates []Candidate) []Candidate {
	seen := map[string]bool{}
	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := candidate.RuleID + "\x00" + candidate.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path == result[j].Path {
			return result[i].RuleID < result[j].RuleID
		}
		return result[i].Path < result[j].Path
	})
	return result
}
