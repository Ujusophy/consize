package securitypolicy

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

func ValidateReport(policy Policy, report FindingReport, now time.Time) error {
	var problems []string
	if err := ValidatePolicy(policy); err != nil {
		return err
	}
	if report.Schema != "./findings.schema.json" {
		problems = append(problems, `$schema must equal "./findings.schema.json"`)
	}
	if report.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Sprintf("schema_version must equal %d", SchemaVersion))
	}
	if !namePattern.MatchString(report.Scanner) {
		problems = append(problems, "scanner is invalid")
	}
	if len(report.ScanID) < 8 {
		problems = append(problems, "scan_id must contain at least eight characters")
	}
	generatedAt, err := time.Parse(time.RFC3339, report.GeneratedAt)
	if err != nil {
		problems = append(problems, "generated_at must be RFC3339")
	} else if generatedAt.After(now.Add(5 * time.Minute)) {
		problems = append(problems, "generated_at must not be in the future")
	}
	if !oneOf(report.Status, "completed", "failed", "timed_out", "data_unavailable", "invalid_report") {
		problems = append(problems, "status is invalid")
	}
	if report.Attempts < 1 || report.Attempts > 2 {
		problems = append(problems, "attempts must be one initial attempt or one bounded retry")
	}
	if len(strings.TrimSpace(report.Target.Component)) < 2 || broad(report.Target.Component) || report.Target.Version == "" || broad(report.Target.Version) {
		problems = append(problems, "target requires component, environment, context, and version")
	}
	if !contains(validEnvironments, report.Target.Environment) {
		problems = append(problems, "target environment is invalid")
	}
	if !contains(validContexts, report.Target.Context) {
		problems = append(problems, "target context is invalid")
	}
	if report.Target.ArtifactDigest != "" && !digestPattern.MatchString(report.Target.ArtifactDigest) {
		problems = append(problems, "target artifact_digest is invalid")
	}
	seen := map[string]bool{}
	for index, finding := range report.Findings {
		prefix := fmt.Sprintf("findings[%d]", index)
		if len(finding.Fingerprint) < 8 {
			problems = append(problems, prefix+": fingerprint must contain at least eight characters")
		}
		key := finding.RuleID + "|" + finding.Fingerprint
		if seen[key] {
			problems = append(problems, prefix+": duplicate rule and fingerprint")
		}
		seen[key] = true
		if strings.TrimSpace(finding.RuleID) == "" {
			problems = append(problems, prefix+": rule_id is required")
		}
		if _, ok := policy.rule(finding.FindingType); !ok {
			problems = append(problems, prefix+": finding_type is not defined by policy")
		}
		if !contains(validSeverities, finding.Severity) {
			problems = append(problems, prefix+": severity is invalid")
		}
		if !contains(validConfidence, finding.Confidence) {
			problems = append(problems, prefix+": confidence is invalid")
		}
		if !contains(validReachability, finding.Reachability) {
			problems = append(problems, prefix+": reachability is invalid")
		}
		if len(strings.TrimSpace(finding.Component)) < 2 || broad(finding.Component) || finding.Version == "" || broad(finding.Version) {
			problems = append(problems, prefix+": component, component_type, environment, context, and version are required")
		}
		if !contains(validComponentTypes, finding.ComponentType) {
			problems = append(problems, prefix+": component_type is invalid")
		}
		if !contains(validEnvironments, finding.Environment) {
			problems = append(problems, prefix+": environment is invalid")
		}
		if !contains(validContexts, finding.Context) {
			problems = append(problems, prefix+": context is invalid")
		}
		if finding.Path != "" && (broad(finding.Path) || filepath.IsAbs(finding.Path) || strings.Contains(filepath.Clean(finding.Path), "..")) {
			problems = append(problems, prefix+": path must be an exact repository-relative path without traversal or globs")
		}
		if finding.ArtifactDigest != "" && !digestPattern.MatchString(finding.ArtifactDigest) {
			problems = append(problems, prefix+": artifact_digest is invalid")
		}
		if oneOf(finding.FindingType, "runtime_image", "development_image", "plugin_permission", "plugin_revoked", "plugin_untrusted_signer") && finding.ArtifactDigest == "" {
			problems = append(problems, prefix+": image and plugin findings require an exact artifact_digest")
		}
		if finding.FindingType == "secret" && !finding.Redacted {
			problems = append(problems, prefix+": secret findings must be redacted")
		}
		if finding.LiveCredential && finding.FindingType != "secret" {
			problems = append(problems, prefix+": live_credential is valid only for secret findings")
		}
		if strings.TrimSpace(finding.Summary) == "" {
			problems = append(problems, prefix+": summary is required")
		}
		if finding.RemediationIssue != "" {
			parsed, err := url.Parse(finding.RemediationIssue)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
				problems = append(problems, prefix+": remediation_issue must be an absolute HTTPS URL")
			}
		}
	}
	return joined(problems)
}

func Evaluate(policy Policy, registry ExceptionRegistry, report FindingReport, now time.Time) (Evaluation, error) {
	if err := ValidateExceptions(policy, registry, now, ""); err != nil {
		return Evaluation{}, err
	}
	if err := ValidateReport(policy, report, now); err != nil {
		return Evaluation{}, err
	}
	result := Evaluation{
		SchemaVersion: SchemaVersion,
		PolicyVersion: policy.PolicyVersion,
		ScanID:        report.ScanID,
		Scanner:       report.Scanner,
		Gate:          GatePassed,
		EvaluatedAt:   now.UTC(),
		Decisions:     []Decision{},
	}
	if report.Status != "completed" {
		result.Gate = GateIncomplete
		result.Counts.Incomplete = 1
		result.Decisions = append(result.Decisions, Decision{
			FindingType: "scanner_failure",
			Decision:    DecisionBlock,
			Reason:      "scanner did not produce a completed, valid report after its bounded retry",
			Owner:       "security",
		})
		return result, nil
	}
	for _, finding := range report.Findings {
		rule, _ := policy.rule(finding.FindingType)
		decision := Decision{
			Fingerprint:      finding.Fingerprint,
			RuleID:           finding.RuleID,
			FindingType:      finding.FindingType,
			Severity:         finding.Severity,
			Owner:            rule.Owner,
			RemediationIssue: finding.RemediationIssue,
		}
		if finding.Owner != "" {
			decision.Owner = finding.Owner
		}
		if !blocks(rule, finding) {
			decision.Decision = DecisionTriage
			decision.Reason = "finding is below the blocking threshold and requires assigned triage"
			result.Counts.Triage++
			result.Decisions = append(result.Decisions, decision)
			continue
		}
		exception, ok := matchingException(report.Scanner, rule, finding, registry.Exceptions, now)
		if ok {
			decision.Decision = DecisionExcepted
			decision.Reason = "finding is covered by a valid, active, exact-scope exception"
			decision.ExceptionID = exception.ID
			result.Counts.Excepted++
			result.Decisions = append(result.Decisions, decision)
			continue
		}
		decision.Decision = DecisionBlock
		decision.Reason = blockReason(rule, finding)
		result.Counts.Blocked++
		result.Decisions = append(result.Decisions, decision)
	}
	if result.Counts.Blocked > 0 {
		result.Gate = GateBlocked
	}
	return result, nil
}

func blocks(rule FindingRule, finding Finding) bool {
	if rule.MandatoryOnly && !finding.Mandatory {
		return false
	}
	switch rule.BlockMode {
	case "always":
		return true
	case "severity":
		return contains(rule.BlockSeverities, finding.Severity)
	case "reachable":
		return contains(rule.BlockedReachability, finding.Reachability)
	case "sast_contextual":
		return contains(rule.BlockSeverities, finding.Severity) &&
			contains(rule.BlockedReachability, finding.Reachability) &&
			confidenceAtLeast(finding.Confidence, rule.MinimumConfidence)
	default:
		return true
	}
}

func confidenceAtLeast(value, minimum string) bool {
	rank := map[string]int{"unknown": 0, "low": 1, "medium": 2, "high": 3}
	return rank[value] >= rank[minimum]
}

func matchingException(scanner string, rule FindingRule, finding Finding, exceptions []Exception, now time.Time) (Exception, bool) {
	if finding.LiveCredential || rule.ExceptionMode == "none" || !finding.ExceptionAllowed {
		return Exception{}, false
	}
	for _, exception := range exceptions {
		if exception.Scanner != scanner || exception.FindingType != finding.FindingType || exception.RuleID != finding.RuleID || exception.Fingerprint != finding.Fingerprint {
			continue
		}
		if exception.Scope.Component != finding.Component || exception.Scope.ComponentType != finding.ComponentType || exception.Scope.Environment != finding.Environment || exception.Scope.Context != finding.Context {
			continue
		}
		if exception.Scope.Path != "" && exception.Scope.Path != finding.Path {
			continue
		}
		if exception.Scope.ArtifactDigest != "" && exception.Scope.ArtifactDigest != finding.ArtifactDigest {
			continue
		}
		if !exceptionRiskCoversFinding(exception.RiskClass, finding.Severity, rule.BlockMode) {
			continue
		}
		if !versionMatches(exception.Scope.AffectedVersionRange, finding.Version) {
			continue
		}
		expires, err := time.Parse(time.RFC3339, exception.Timeline.ExpiresAt)
		if err != nil || !expires.After(now) {
			continue
		}
		if rule.ExceptionMode == "false_positive_only" && exception.Disposition != "false_positive" {
			continue
		}
		return exception, true
	}
	return Exception{}, false
}

func exceptionRiskCoversFinding(riskClass, severity, blockMode string) bool {
	rank := map[string]int{"info": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}
	if severity == "unknown" {
		if blockMode == "reachable" {
			return rank[riskClass] >= rank["high"]
		}
		return true
	}
	return rank[riskClass] >= rank[severity]
}

func blockReason(rule FindingRule, finding Finding) string {
	if finding.LiveCredential {
		return "live credential exposure is an incident and is not eligible for an ordinary exception"
	}
	if rule.ExceptionMode == "none" {
		return "policy classifies this control failure as non-exceptionable"
	}
	if !finding.ExceptionAllowed {
		return "the originating rule does not allow a contextual exception"
	}
	return "blocking finding has no valid, active, exact-scope exception"
}
