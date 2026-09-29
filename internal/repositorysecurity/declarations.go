package repositorysecurity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	declarationIDPattern = regexp.MustCompile(`^REP-ALLOW-[0-9]{4}-[0-9]{4}$`)
	digestPattern        = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	principalPattern     = regexp.MustCompile(`^(github:[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})|team:[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)$`)
)

var declarableRules = map[string]bool{
	"private-env-file":             true,
	"credential-file-name":         true,
	"embedded-credential":          true,
	"private-key-content":          true,
	"kubeconfig-content":           true,
	"cloud-credential-content":     true,
	"unexpected-executable-binary": true,
	"unexpected-executable-mode":   true,
	"oversized-generated-file":     true,
}

func LoadDeclarations(path string, now time.Time) (DeclarationRegistry, error) {
	file, err := os.Open(path)
	if err != nil {
		return DeclarationRegistry{}, err
	}
	defer file.Close()

	var registry DeclarationRegistry
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&registry); err != nil {
		return DeclarationRegistry{}, fmt.Errorf("decode declarations: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return DeclarationRegistry{}, fmt.Errorf("declarations contain trailing JSON data")
	}
	if err := ValidateDeclarations(registry, now); err != nil {
		return DeclarationRegistry{}, err
	}
	return registry, nil
}

func ValidateDeclarations(registry DeclarationRegistry, now time.Time) error {
	var problems []string
	if registry.Schema != "./repository-allowlist.schema.json" {
		problems = append(problems, `$schema must equal "./repository-allowlist.schema.json"`)
	}
	if registry.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Sprintf("schema_version must equal %d", SchemaVersion))
	}
	seenIDs := map[string]bool{}
	seenScopes := map[string]bool{}
	for index, declaration := range registry.Declarations {
		prefix := fmt.Sprintf("declarations[%d]", index)
		if declaration.ID != "" {
			prefix = declaration.ID
		}
		for _, problem := range validateDeclaration(declaration, now) {
			problems = append(problems, prefix+": "+problem)
		}
		if seenIDs[declaration.ID] {
			problems = append(problems, prefix+": duplicate declaration id")
		}
		seenIDs[declaration.ID] = true
		scope := declaration.RuleID + "|" + declaration.Path
		if seenScopes[scope] {
			problems = append(problems, prefix+": duplicate rule and path declaration")
		}
		seenScopes[scope] = true
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("repository declaration validation failed:\n- %s", strings.Join(problems, "\n- "))
}

func validateDeclaration(declaration Declaration, now time.Time) []string {
	var problems []string
	if !declarationIDPattern.MatchString(declaration.ID) {
		problems = append(problems, "id must match REP-ALLOW-YYYY-NNNN")
	}
	if !declarableRules[declaration.RuleID] {
		problems = append(problems, "rule_id is not eligible for a repository declaration")
	}
	if !exactRelativePath(declaration.Path) {
		problems = append(problems, "path must be exact and repository-relative")
	}
	if !digestPattern.MatchString(declaration.SHA256) {
		problems = append(problems, "sha256 must contain an exact lower-case file digest")
	}
	if !oneOf(declaration.Classification, "example", "test_fixture", "generated_asset", "approved_tool") {
		problems = append(problems, "classification is invalid")
	}
	if declaration.Classification == "example" && !isExamplePath(declaration.Path) {
		problems = append(problems, "example declarations require an example, sample, or template filename")
	}
	if declaration.Classification == "test_fixture" && !isFixturePath(declaration.Path) {
		problems = append(problems, "test_fixture declarations must be under a testdata or fixtures directory")
	}
	if len(strings.TrimSpace(declaration.Reason)) < 20 {
		problems = append(problems, "reason must contain at least 20 characters")
	}
	if !principalPattern.MatchString(declaration.Owner) {
		problems = append(problems, "owner must be a github:USER or team:ORG/TEAM principal")
	}
	if !principalPattern.MatchString(declaration.ApprovedBy) {
		problems = append(problems, "approved_by must be a github:USER or team:ORG/TEAM principal")
	}
	if declaration.Owner == declaration.ApprovedBy {
		problems = append(problems, "approved_by must be independent of the declaration owner")
	}
	reviewed, reviewErr := time.Parse(time.RFC3339, declaration.ReviewedAt)
	expires, expiryErr := time.Parse(time.RFC3339, declaration.ExpiresAt)
	if reviewErr != nil {
		problems = append(problems, "reviewed_at must be RFC3339")
	}
	if expiryErr != nil {
		problems = append(problems, "expires_at must be RFC3339")
	}
	if reviewErr == nil && expiryErr == nil {
		if reviewed.After(now.Add(5 * time.Minute)) {
			problems = append(problems, "reviewed_at must not be in the future")
		}
		if !expires.After(now) {
			problems = append(problems, "declaration is expired")
		}
		if !expires.After(reviewed) {
			problems = append(problems, "expires_at must be after reviewed_at")
		}
		if expires.Sub(reviewed) > 366*24*time.Hour {
			problems = append(problems, "declaration duration must not exceed one year")
		}
	}
	return problems
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func exactRelativePath(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return path != "" && clean == path && clean != "." && !filepath.IsAbs(path) && !strings.HasPrefix(clean, "../") && !strings.ContainsAny(path, "*?[]{}")
}

func isExamplePath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, ".example") || strings.Contains(base, ".sample") || strings.Contains(base, ".template")
}

func isFixturePath(path string) bool {
	path = "/" + strings.ToLower(filepath.ToSlash(path)) + "/"
	return strings.Contains(path, "/testdata/") || strings.Contains(path, "/fixtures/")
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
