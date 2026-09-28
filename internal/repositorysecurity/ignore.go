package repositorysecurity

import (
	"bufio"
	"path/filepath"
	"strings"
)

type ignoreRequirement struct {
	name         string
	alternatives []string
}

var gitIgnoreRequirements = []ignoreRequirement{
	{name: "private environment files", alternatives: []string{".env"}},
	{name: "private environment variants", alternatives: []string{".env.*"}},
	{name: "Consize local state", alternatives: []string{".consize/", ".consize"}},
	{name: "Go build cache", alternatives: []string{".gocache/", ".gocache"}},
	{name: "audit and JSONL logs", alternatives: []string{"*.jsonl"}},
	{name: "general logs", alternatives: []string{"*.log"}},
	{name: "frontend dependencies", alternatives: []string{"ui/node_modules/", "ui/node_modules"}},
	{name: "frontend build output", alternatives: []string{"ui/.next/", "ui/.next"}},
	{name: "frontend coverage", alternatives: []string{"ui/coverage/", "ui/coverage", "coverage/", "coverage"}},
	{name: "repository coverage", alternatives: []string{"coverage/", "coverage"}},
	{name: "downloaded security tools", alternatives: []string{".tools/", ".tools", ".security-tools/", ".security-tools"}},
	{name: "security scanner caches", alternatives: []string{".cache/security/", ".cache/security", ".trivycache/", ".trivycache"}},
	{name: "temporary output", alternatives: []string{"tmp/", "tmp"}},
	{name: "generated output", alternatives: []string{"output/", "output"}},
	{name: "binary output", alternatives: []string{"bin/", "bin"}},
	{name: "distribution output", alternatives: []string{"dist/", "dist"}},
	{name: "Go test binaries", alternatives: []string{"*.test"}},
	{name: "documentation build output", alternatives: []string{"site/", "site"}},
}

var dockerIgnoreRequirements = []ignoreRequirement{
	{name: "Git metadata", alternatives: []string{".git", ".git/"}},
	{name: "private environment files", alternatives: []string{".env"}},
	{name: "private environment variants", alternatives: []string{".env.*"}},
	{name: "Consize local state", alternatives: []string{".consize", ".consize/"}},
	{name: "audit and JSONL logs", alternatives: []string{"*.jsonl", "**/*.jsonl"}},
	{name: "frontend dependencies", alternatives: []string{"ui/node_modules", "ui/node_modules/"}},
	{name: "frontend build output", alternatives: []string{"ui/.next", "ui/.next/"}},
	{name: "downloaded security tools", alternatives: []string{".tools", ".tools/", ".security-tools", ".security-tools/"}},
}

func scanIgnoreFiles(files map[string]TrackedFile) []Candidate {
	var findings []Candidate
	gitignore, ok := files[".gitignore"]
	if !ok || !gitignore.ContentSet {
		findings = append(findings, configurationCandidate("missing-gitignore", ".gitignore", "repository must provide a tracked .gitignore", "high"))
	} else {
		findings = append(findings, missingIgnoreRequirements(".gitignore", gitignore.Content, gitIgnoreRequirements)...)
	}

	if containsContainerDefinition(files) {
		dockerignore, ok := files[".dockerignore"]
		if !ok || !dockerignore.ContentSet {
			findings = append(findings, configurationCandidate("missing-dockerignore", ".dockerignore", "container build files require a tracked .dockerignore", "high"))
		} else {
			findings = append(findings, missingIgnoreRequirements(".dockerignore", dockerignore.Content, dockerIgnoreRequirements)...)
		}
	}
	return findings
}

func missingIgnoreRequirements(path string, content []byte, requirements []ignoreRequirement) []Candidate {
	patterns := parseIgnorePatterns(content)
	var findings []Candidate
	for _, requirement := range requirements {
		present := false
		for _, alternative := range requirement.alternatives {
			if patterns[normalizeIgnorePattern(alternative)] {
				present = true
				break
			}
		}
		if !present {
			findings = append(findings, configurationCandidate("missing-ignore-rule", path, "ignore file does not protect "+requirement.name, "high"))
		}
	}
	return findings
}

func parseIgnorePatterns(content []byte) map[string]bool {
	patterns := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		patterns[normalizeIgnorePattern(line)] = true
	}
	return patterns
}

func normalizeIgnorePattern(pattern string) string {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	pattern = strings.TrimPrefix(pattern, "/")
	return pattern
}

func containsContainerDefinition(files map[string]TrackedFile) bool {
	for path := range files {
		base := strings.ToLower(filepath.Base(path))
		if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") || base == "compose.yml" || base == "compose.yaml" || strings.HasPrefix(base, "docker-compose.") {
			return true
		}
	}
	return false
}

func generatedOrLocalPath(path string) bool {
	path = "/" + strings.ToLower(filepath.ToSlash(path)) + "/"
	base := strings.ToLower(filepath.Base(strings.TrimSuffix(path, "/")))
	return containsAny(path,
		"/.consize/",
		"/.gocache/",
		"/.next/",
		"/node_modules/",
		"/coverage/",
		"/bin/",
		"/dist/",
		"/.tools/",
		"/.security-tools/",
		"/.cache/security/",
		"/.trivycache/",
	) || strings.HasSuffix(base, ".jsonl") || strings.HasSuffix(base, ".test")
}
