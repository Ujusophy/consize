package repositorysecurity

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	awsAccessKeyPattern = regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`)
	githubTokenPattern  = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`)
	genericAssignment   = regexp.MustCompile(`(?i)\b([A-Z0-9_-]*(?:password|passwd|secret|token|api[_-]?key|client[_-]?secret|private[_-]?key|access[_-]?key)[A-Z0-9_-]*)["']?\s*[:=]\s*["']?([^"'\s,#}]{8,})`)
	publicVariable      = regexp.MustCompile(`\bNEXT_PUBLIC_[A-Z0-9_]+\b`)
	processEnvironment  = regexp.MustCompile(`process\.env(?:\.([A-Z][A-Z0-9_]*)|\[["']([A-Z][A-Z0-9_]*)["']\])`)
	importPathPattern   = regexp.MustCompile(`(?m)\bfrom\s+["']([^"']+)["']|\brequire\(["']([^"']+)["']\)`)
)

var sensitiveNameParts = []string{
	"SECRET",
	"TOKEN",
	"PASSWORD",
	"PASSWD",
	"PRIVATE_KEY",
	"CREDENTIAL",
	"KUBECONFIG",
	"DATABASE_URL",
	"ACCESS_KEY",
}

var privateKeyMarkers = []string{
	"-----BEGIN " + "PRIVATE KEY-----",
	"-----BEGIN " + "RSA PRIVATE KEY-----",
	"-----BEGIN " + "EC PRIVATE KEY-----",
	"-----BEGIN " + "OPENSSH PRIVATE KEY-----",
}

func scanContent(file TrackedFile) []Candidate {
	if !file.ContentSet {
		return nil
	}
	var findings []Candidate
	content := file.Content
	lowerPath := strings.ToLower(file.Path)
	base := strings.ToLower(filepath.Base(file.Path))

	if privateEnvironmentFile(base) {
		findings = append(findings, secretCandidate("private-env-file", file.Path, "tracked private environment file is prohibited"))
	}
	if credentialFilename(lowerPath, base) {
		findings = append(findings, secretCandidate("credential-file-name", file.Path, "tracked filename is reserved for credential or private-key material"))
	}
	if credentialArchive(lowerPath, base) {
		findings = append(findings, secretCandidate("credential-archive", file.Path, "tracked archive name indicates bundled credential material"))
	}

	if binaryKind(content) != "" {
		findings = append(findings, configurationCandidate("unexpected-executable-binary", file.Path, "tracked file contains an executable binary format", "critical"))
		return findings
	}
	if looksBinary(content) {
		return findings
	}

	text := string(content)
	if containsAny(text, privateKeyMarkers...) {
		findings = append(findings, secretCandidate("private-key-content", file.Path, "tracked file contains private-key material"))
	}
	if looksLikeKubeconfig(text) {
		findings = append(findings, secretCandidate("kubeconfig-content", file.Path, "tracked file contains a Kubernetes client configuration"))
	}
	if awsAccessKeyPattern.MatchString(text) || githubTokenPattern.MatchString(text) || looksLikeCloudCredential(text) {
		findings = append(findings, secretCandidate("cloud-credential-content", file.Path, "tracked file contains cloud or source-control credential material"))
	}
	if hasEmbeddedCredential(file.Path, text) {
		findings = append(findings, secretCandidate("embedded-credential", file.Path, "tracked file contains a non-placeholder credential assignment"))
	}

	findings = append(findings, scanFrontendExposure(file.Path, text)...)
	return findings
}

func privateEnvironmentFile(base string) bool {
	return base == ".env" || strings.HasPrefix(base, ".env.")
}

func credentialFilename(path, base string) bool {
	extension := strings.ToLower(filepath.Ext(base))
	if oneOf(extension, ".pem", ".key", ".p12", ".pfx", ".jks", ".keystore", ".kubeconfig") {
		return true
	}
	if oneOf(base, "id_rsa", "id_ed25519", "id_ecdsa", "kubeconfig", "credentials", "credentials.json", "service-account.json", "service_account.json", "application_default_credentials.json") {
		return true
	}
	return strings.Contains(path, "/.kube/config") || strings.Contains(path, "/.aws/credentials")
}

func credentialArchive(path, base string) bool {
	archive := strings.HasSuffix(base, ".zip") || strings.HasSuffix(base, ".tar") || strings.HasSuffix(base, ".tar.gz") || strings.HasSuffix(base, ".tgz") || strings.HasSuffix(base, ".7z")
	if !archive {
		return false
	}
	return containsAny(strings.ToLower(path), "credential", "secret", "private-key", "private_key", "kubeconfig", "service-account", "service_account")
}

func looksLikeKubeconfig(text string) bool {
	return strings.Contains(text, "apiVersion: "+"v1") && strings.Contains(text, "clus"+"ters:") && strings.Contains(text, "con"+"texts:") && strings.Contains(text, "current-"+"context:")
}

func looksLikeCloudCredential(text string) bool {
	lower := strings.ToLower(text)
	google := strings.Contains(lower, `"ty`+`pe"`) && strings.Contains(lower, `"service_`+`account"`) && strings.Contains(lower, `"private_`+`key"`)
	azure := strings.Contains(lower, "account"+"key=") || strings.Contains(lower, "client"+"secret=")
	return google || azure
}

func hasEmbeddedCredential(path, text string) bool {
	for _, match := range genericAssignment.FindAllStringSubmatch(text, -1) {
		if len(match) < 3 || placeholder(match[2]) {
			continue
		}
		key := strings.ToUpper(match[1])
		if strings.HasSuffix(key, "_ENV") || strings.HasSuffix(key, "ENV") {
			continue
		}
		if strings.HasPrefix(match[2], "[]") || strings.HasPrefix(match[2], "regexp.MustCompile") {
			continue
		}
		if strings.HasSuffix(path, "_test.go") && containsAny(strings.ToLower(match[2]), "test-token", "viewer-token", "operator-token", "fixture") {
			continue
		}
		return true
	}
	return false
}

func placeholder(value string) bool {
	value = strings.TrimSpace(strings.Trim(value, `"'`))
	lower := strings.ToLower(value)
	if value == "" || strings.Contains(value, "${") || strings.Contains(value, "{{") || strings.Contains(value, "...") || strings.HasPrefix(value, "<") {
		return true
	}
	return containsAny(lower, "example", "placeholder", "change-me", "changeme", "replace-me", "replace_me", "your-", "your_", "dummy", "fake", "redacted", "test-only", "not-a-secret")
}

func scanFrontendExposure(path, text string) []Candidate {
	if !frontendSource(path) {
		return nil
	}
	var findings []Candidate
	for _, variable := range publicVariable.FindAllString(text, -1) {
		if sensitiveConfigurationName(variable) {
			findings = append(findings, secretCandidate("public-sensitive-variable", path, "browser-exposed environment variable name indicates server-only configuration"))
		}
	}
	client := hasClientDirective(text)
	if client {
		for _, match := range processEnvironment.FindAllStringSubmatch(text, -1) {
			name := match[1]
			if name == "" {
				name = match[2]
			}
			if !strings.HasPrefix(name, "NEXT_PUBLIC_") || sensitiveConfigurationName(name) {
				findings = append(findings, secretCandidate("server-config-in-client", path, "client component reads server-only environment configuration"))
				break
			}
		}
		for _, match := range importPathPattern.FindAllStringSubmatch(text, -1) {
			importPath := match[1]
			if importPath == "" {
				importPath = match[2]
			}
			if serverOnlyImport(importPath) {
				findings = append(findings, secretCandidate("server-config-client-import", path, "client component imports a server-only configuration module"))
				break
			}
		}
	}
	if (strings.Contains(text, "publicRuntimeConfig") || strings.Contains(text, "publicConfig")) && containsSensitiveEnvironmentReference(text) {
		findings = append(findings, secretCandidate("server-config-public-object", path, "public configuration object references server-only configuration"))
	}
	return findings
}

func frontendSource(path string) bool {
	path = strings.ToLower(path)
	return strings.HasPrefix(path, "ui/") && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".jsx") || strings.HasSuffix(path, ".mjs") || strings.HasSuffix(path, ".cjs"))
}

func hasClientDirective(text string) bool {
	prefix := text
	if len(prefix) > 512 {
		prefix = prefix[:512]
	}
	return strings.Contains(prefix, `"use client"`) || strings.Contains(prefix, `'use client'`)
}

func sensitiveConfigurationName(name string) bool {
	name = strings.ToUpper(name)
	for _, part := range sensitiveNameParts {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

func containsSensitiveEnvironmentReference(text string) bool {
	for _, match := range processEnvironment.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if name == "" {
			name = match[2]
		}
		if sensitiveConfigurationName(name) {
			return true
		}
	}
	return false
}

func serverOnlyImport(path string) bool {
	path = "/" + strings.ToLower(filepath.ToSlash(path)) + "/"
	return containsAny(path, "/server/", "/server-only/", "/secret/", "/secrets/", "/private/", "/backend/")
}

func binaryKind(content []byte) string {
	if len(content) >= 4 {
		signature := content[:4]
		switch {
		case bytes.Equal(signature, []byte{0x7f, 'E', 'L', 'F'}):
			return "elf"
		case bytes.Equal(signature, []byte{0x00, 'a', 's', 'm'}):
			return "wasm"
		case bytes.Equal(signature, []byte{0xfe, 0xed, 0xfa, 0xce}), bytes.Equal(signature, []byte{0xce, 0xfa, 0xed, 0xfe}), bytes.Equal(signature, []byte{0xfe, 0xed, 0xfa, 0xcf}), bytes.Equal(signature, []byte{0xcf, 0xfa, 0xed, 0xfe}), bytes.Equal(signature, []byte{0xca, 0xfe, 0xba, 0xbe}):
			return "mach-o"
		}
	}
	if len(content) >= 2 && content[0] == 'M' && content[1] == 'Z' {
		return "pe"
	}
	return ""
}

func looksBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	sample := content
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	return bytes.IndexByte(sample, 0) >= 0
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

func secretCandidate(ruleID, path, summary string) Candidate {
	return Candidate{RuleID: ruleID, Path: path, FindingType: "secret", Severity: "critical", ComponentType: "configuration", Summary: summary, ExceptionAllowed: true}
}

func configurationCandidate(ruleID, path, summary, severity string) Candidate {
	return Candidate{RuleID: ruleID, Path: path, FindingType: "configuration", Severity: severity, ComponentType: "configuration", Summary: summary, ExceptionAllowed: true}
}
