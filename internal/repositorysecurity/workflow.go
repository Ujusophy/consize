package repositorysecurity

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	fullSHA       = regexp.MustCompile(`^[a-f0-9]{40}$`)
	untrustedExpr = regexp.MustCompile(`\$\{\{\s*github\.event\.(?:pull_request\.(?:title|body)|issue\.(?:title|body)|comment\.body|head_commit\.message)`)
)

type workflowDocument struct {
	Permissions any                    `yaml:"permissions"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	Permissions any            `yaml:"permissions"`
	Steps       []workflowStep `yaml:"steps"`
}

type workflowStep struct {
	Uses string `yaml:"uses"`
	Run  string `yaml:"run"`
}

func scanWorkflow(file TrackedFile) []Candidate {
	if !file.ContentSet || !strings.HasPrefix(file.Path, ".github/workflows/") {
		return nil
	}
	var raw yaml.Node
	if err := yaml.Unmarshal(file.Content, &raw); err != nil {
		return []Candidate{workflowCandidate("invalid-workflow-yaml", file.Path, "workflow YAML cannot be parsed", "critical")}
	}
	if workflowEvent(raw.Content[0], "pull_request_target") {
		return []Candidate{workflowCandidate("pull-request-target", file.Path, "pull_request_target workflows are prohibited because they combine privileged context with untrusted contributions", "critical")}
	}
	var workflow workflowDocument
	if err := yaml.Unmarshal(file.Content, &workflow); err != nil {
		return []Candidate{workflowCandidate("invalid-workflow-yaml", file.Path, "workflow structure cannot be parsed", "critical")}
	}
	var findings []Candidate
	if workflow.Permissions == nil {
		findings = append(findings, workflowCandidate("missing-workflow-permissions", file.Path, "workflow must declare least-privilege top-level permissions", "high"))
	}
	if permissionWrites(workflow.Permissions) && workflowEvent(raw.Content[0], "pull_request") {
		findings = append(findings, workflowCandidate("pr-write-permissions", file.Path, "pull request workflow grants write permissions", "critical"))
	}
	for _, job := range workflow.Jobs {
		if permissionWrites(job.Permissions) && workflowEvent(raw.Content[0], "pull_request") {
			findings = append(findings, workflowCandidate("pr-write-permissions", file.Path, "pull request job grants write permissions", "critical"))
		}
		for _, step := range job.Steps {
			if step.Uses != "" && !trustedActionReference(step.Uses) {
				findings = append(findings, workflowCandidate("mutable-action-reference", file.Path, "workflow action reference is not pinned to an immutable commit SHA", "high"))
			}
			if untrustedExpr.MatchString(step.Run) {
				findings = append(findings, workflowCandidate("untrusted-expression-in-shell", file.Path, "untrusted event content is interpolated directly into a shell command", "critical"))
			}
		}
	}
	return findings
}

func workflowEvent(root *yaml.Node, name string) bool {
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value != "on" {
			continue
		}
		node := root.Content[index+1]
		if node.Kind == yaml.ScalarNode {
			return node.Value == name
		}
		if node.Kind == yaml.SequenceNode {
			for _, child := range node.Content {
				if child.Value == name {
					return true
				}
			}
		}
		if node.Kind == yaml.MappingNode {
			for index := 0; index+1 < len(node.Content); index += 2 {
				if node.Content[index].Value == name {
					return true
				}
			}
		}
	}
	return false
}

func permissionWrites(value any) bool {
	if value == nil {
		return false
	}
	if scalar, ok := value.(string); ok {
		return scalar == "write-all"
	}
	if permissions, ok := value.(map[string]any); ok {
		for _, access := range permissions {
			if access == "write" {
				return true
			}
		}
	}
	return false
}

func trustedActionReference(reference string) bool {
	if strings.HasPrefix(reference, "./") {
		return true
	}
	if strings.HasPrefix(reference, "docker://") {
		parts := strings.Split(reference, "@sha256:")
		return len(parts) == 2 && len(parts[1]) == 64
	}
	_, version, ok := strings.Cut(reference, "@")
	return ok && fullSHA.MatchString(version)
}

func workflowCandidate(ruleID, path, summary, severity string) Candidate {
	return Candidate{RuleID: ruleID, Path: path, FindingType: "ci_workflow", Severity: severity, ComponentType: "workflow", Summary: summary, ExceptionAllowed: true}
}
