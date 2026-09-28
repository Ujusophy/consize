package securitypolicy

import (
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	policyVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	exceptionIDPattern   = regexp.MustCompile(`^SEC-EXC-[0-9]{4}-[0-9]{4}$`)
	namePattern          = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
	principalPattern     = regexp.MustCompile(`^(github:[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})|team:[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)$`)
	digestPattern        = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

var requiredFindingTypes = []string{
	"secret",
	"frontend_dependency",
	"go_vulnerability",
	"runtime_image",
	"development_image",
	"configuration",
	"sast",
	"ci_workflow",
	"license",
	"dependency_source",
	"sbom_failure",
	"release_integrity",
	"kubernetes_iac",
	"plugin_permission",
	"plugin_revoked",
	"plugin_untrusted_signer",
	"runtime_eol",
	"security_regression",
	"report_delivery",
	"report_schema",
	"scanner_failure",
}

var (
	validSeverities     = []string{"critical", "high", "medium", "low", "info", "unknown"}
	validConfidence     = []string{"high", "medium", "low", "unknown"}
	validReachability   = []string{"reachable", "not_reachable", "unknown", "not_applicable"}
	validReleaseImpacts = []string{"block", "release_owner_review", "track"}
	validOwners         = []string{"backend", "frontend", "integration", "plugin", "security", "release"}
	validEnvironments   = []string{"development", "test", "ci", "staging", "production", "release"}
	validContexts       = []string{"pull_request", "default_branch", "scheduled_scan", "development_image", "runtime_image", "build", "release", "plugin_install", "deployment"}
	validComponentTypes = []string{"source", "dependency", "runtime_image", "development_image", "workflow", "configuration", "kubernetes", "iac", "plugin", "artifact", "runtime", "report"}
)

var nonExceptionableTypes = map[string]bool{
	"sbom_failure":            true,
	"release_integrity":       true,
	"plugin_revoked":          true,
	"plugin_untrusted_signer": true,
	"security_regression":     true,
	"report_delivery":         true,
	"report_schema":           true,
	"scanner_failure":         true,
}

func ValidatePolicy(policy Policy) error {
	var problems []string
	if policy.Schema != "./policy.schema.json" {
		problems = append(problems, `$schema must equal "./policy.schema.json"`)
	}
	if policy.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Sprintf("schema_version must equal %d", SchemaVersion))
	}
	if !policyVersionPattern.MatchString(policy.PolicyVersion) {
		problems = append(problems, "policy_version must be numeric semantic version")
	}
	for _, severity := range []string{"critical", "high", "medium", "low"} {
		limit, ok := policy.ExceptionLimits[severity]
		if !ok {
			problems = append(problems, "exception_limits missing "+severity)
		} else {
			if limit.MaxDurationHours < 1 || limit.ReviewIntervalHours < 1 {
				problems = append(problems, "exception limits for "+severity+" must be positive")
			}
			if limit.ReviewIntervalHours > limit.MaxDurationHours {
				problems = append(problems, "review interval for "+severity+" exceeds maximum duration")
			}
			if len(limit.ApproverRoles) == 0 {
				problems = append(problems, "approver roles for "+severity+" are required")
			}
			for _, role := range limit.ApproverRoles {
				if !oneOf(role, "security", "repository_owner") {
					problems = append(problems, "exception limits for "+severity+" contain an invalid approver role")
				}
			}
			if duplicate := duplicateValue(limit.ApproverRoles); duplicate != "" {
				problems = append(problems, "exception limits for "+severity+" contain duplicate approver role "+duplicate)
			}
		}
		service, ok := policy.ServiceLevels[severity]
		if !ok {
			problems = append(problems, "service_levels missing "+severity)
		} else if service.AcknowledgementHours < 1 || service.RemediationDecisionHours < 1 {
			problems = append(problems, "service levels for "+severity+" must be positive")
		} else if !contains(validReleaseImpacts, service.ReleaseImpact) {
			problems = append(problems, "service levels for "+severity+" contain an invalid release impact")
		}
	}
	if len(policy.ExceptionLimits) != 4 {
		problems = append(problems, "exception_limits contains unsupported severity keys")
	}
	if len(policy.ServiceLevels) != 4 {
		problems = append(problems, "service_levels contains unsupported severity keys")
	}

	rules := make(map[string]FindingRule, len(policy.FindingRules))
	for _, rule := range policy.FindingRules {
		if _, exists := rules[rule.FindingType]; exists {
			problems = append(problems, "duplicate finding rule: "+rule.FindingType)
			continue
		}
		rules[rule.FindingType] = rule
		if len(strings.TrimSpace(rule.Description)) < 10 {
			problems = append(problems, "finding rule "+rule.FindingType+" requires a meaningful description")
		}
		if !oneOf(rule.BlockMode, "always", "severity", "reachable", "sast_contextual") {
			problems = append(problems, "finding rule "+rule.FindingType+" has invalid block_mode")
		}
		if !oneOf(rule.ExceptionMode, "none", "false_positive_only", "risk_or_false_positive", "contextual") {
			problems = append(problems, "finding rule "+rule.FindingType+" has invalid exception_mode")
		}
		if (rule.BlockMode == "severity" || rule.BlockMode == "sast_contextual") && len(rule.BlockSeverities) == 0 {
			problems = append(problems, "finding rule "+rule.FindingType+" requires block_severities")
		}
		for _, severity := range rule.BlockSeverities {
			if !contains(validSeverities, severity) {
				problems = append(problems, "finding rule "+rule.FindingType+" contains an invalid blocking severity")
			}
		}
		if duplicate := duplicateValue(rule.BlockSeverities); duplicate != "" {
			problems = append(problems, "finding rule "+rule.FindingType+" contains duplicate blocking severity "+duplicate)
		}
		if (rule.BlockMode == "reachable" || rule.BlockMode == "sast_contextual") && len(rule.BlockedReachability) == 0 {
			problems = append(problems, "finding rule "+rule.FindingType+" requires blocked_reachability")
		}
		for _, reachability := range rule.BlockedReachability {
			if !contains(validReachability, reachability) {
				problems = append(problems, "finding rule "+rule.FindingType+" contains invalid reachability")
			}
		}
		if duplicate := duplicateValue(rule.BlockedReachability); duplicate != "" {
			problems = append(problems, "finding rule "+rule.FindingType+" contains duplicate reachability "+duplicate)
		}
		if rule.BlockMode == "sast_contextual" && !contains(validConfidence, rule.MinimumConfidence) {
			problems = append(problems, "finding rule "+rule.FindingType+" requires a valid minimum_confidence")
		}
		if nonExceptionableTypes[rule.FindingType] && rule.ExceptionMode != "none" {
			problems = append(problems, "finding rule "+rule.FindingType+" must not permit exceptions")
		}
		if rule.RequiredExceptionEnvironment != "" && !contains(validEnvironments, rule.RequiredExceptionEnvironment) {
			problems = append(problems, "finding rule "+rule.FindingType+" has an invalid exception environment")
		}
		if !contains(validReleaseImpacts, rule.ReleaseImpact) {
			problems = append(problems, "finding rule "+rule.FindingType+" has an invalid release impact")
		}
		if !contains(validOwners, rule.Owner) {
			problems = append(problems, "finding rule "+rule.FindingType+" has an invalid owner")
		}
	}
	for _, findingType := range requiredFindingTypes {
		if _, ok := rules[findingType]; !ok {
			problems = append(problems, "finding_rules missing "+findingType)
		}
	}
	if len(rules) != len(requiredFindingTypes) {
		problems = append(problems, "finding_rules contains an unsupported finding type")
	}
	return joined(problems)
}

func ValidateExceptions(policy Policy, registry ExceptionRegistry, now time.Time, changeAuthor string) error {
	var problems []string
	if err := ValidatePolicy(policy); err != nil {
		return err
	}
	if registry.Schema != "./exceptions.schema.json" {
		problems = append(problems, `$schema must equal "./exceptions.schema.json"`)
	}
	if registry.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Sprintf("schema_version must equal %d", SchemaVersion))
	}
	if registry.PolicyVersion != policy.PolicyVersion {
		problems = append(problems, "exception registry policy_version does not match policy")
	}
	seenIDs := map[string]bool{}
	seenScopes := map[string]string{}
	for index, exception := range registry.Exceptions {
		prefix := fmt.Sprintf("exceptions[%d]", index)
		if exception.ID != "" {
			prefix = exception.ID
		}
		entryProblems := validateException(policy, exception, now, changeAuthor)
		for _, problem := range entryProblems {
			problems = append(problems, prefix+": "+problem)
		}
		if seenIDs[exception.ID] {
			problems = append(problems, prefix+": duplicate exception id")
		}
		seenIDs[exception.ID] = true
		key := strings.Join([]string{
			exception.Scanner,
			exception.FindingType,
			exception.RuleID,
			exception.Fingerprint,
			exception.Scope.Component,
			exception.Scope.Environment,
			exception.Scope.Context,
			exception.Scope.AffectedVersionRange,
		}, "|")
		if existing := seenScopes[key]; existing != "" {
			problems = append(problems, prefix+": duplicates finding scope already owned by "+existing)
		}
		seenScopes[key] = exception.ID
	}
	return joined(problems)
}

func ValidateExceptionChanges(baseline, current ExceptionRegistry, changeAuthor string) error {
	if changeAuthor == "" {
		return nil
	}
	baselineByID := make(map[string]Exception, len(baseline.Exceptions))
	for _, exception := range baseline.Exceptions {
		baselineByID[exception.ID] = exception
	}
	var problems []string
	for _, exception := range current.Exceptions {
		previous, existed := baselineByID[exception.ID]
		if existed && reflect.DeepEqual(previous, exception) {
			continue
		}
		if oneOf(exception.RiskClass, "critical", "high") && normalizePrincipal(changeAuthor) == exception.Ownership.ApprovedBy {
			problems = append(problems, exception.ID+": high and critical exceptions cannot be approved by the change author")
		}
	}
	return joined(problems)
}

func validateException(policy Policy, exception Exception, now time.Time, changeAuthor string) []string {
	var problems []string
	if !exceptionIDPattern.MatchString(exception.ID) {
		problems = append(problems, "id must match SEC-EXC-YYYY-NNNN")
	}
	if !namePattern.MatchString(exception.Scanner) || broad(exception.Scanner) {
		problems = append(problems, "scanner must be an exact lower-case scanner id")
	}
	if strings.TrimSpace(exception.RuleID) == "" || broad(exception.RuleID) {
		problems = append(problems, "rule_id must be exact and must not contain glob syntax")
	}
	if len(exception.Fingerprint) < 8 || broad(exception.Fingerprint) {
		problems = append(problems, "fingerprint must be an exact stable value of at least eight characters")
	}
	if !oneOf(exception.RiskClass, "critical", "high", "medium", "low") {
		problems = append(problems, "risk_class is invalid")
	}
	if !oneOf(exception.Disposition, "risk_acceptance", "false_positive") {
		problems = append(problems, "disposition is invalid")
	}
	rule, ok := policy.rule(exception.FindingType)
	if !ok {
		problems = append(problems, "finding_type is not defined by policy")
	} else {
		switch rule.ExceptionMode {
		case "none":
			problems = append(problems, "finding type is not exceptionable")
		case "false_positive_only":
			if exception.Disposition != "false_positive" {
				problems = append(problems, "finding type permits only a narrow false-positive disposition")
			}
		}
		if rule.RequiredExceptionEnvironment != "" && exception.Scope.Environment != rule.RequiredExceptionEnvironment {
			problems = append(problems, "exception environment must be "+rule.RequiredExceptionEnvironment)
		}
	}
	if exception.Disposition == "false_positive" && len(strings.TrimSpace(exception.FalsePositiveEvidence)) < 20 {
		problems = append(problems, "false_positive_evidence is required for a false-positive disposition")
	}
	if exception.Disposition != "false_positive" && exception.FalsePositiveEvidence != "" {
		problems = append(problems, "false_positive_evidence is only valid for a false-positive disposition")
	}
	if len(strings.TrimSpace(exception.Reason)) < 20 {
		problems = append(problems, "reason must contain at least 20 characters")
	}
	if len(exception.CompensatingControls) == 0 {
		problems = append(problems, "at least one compensating control is required")
	}
	for _, control := range exception.CompensatingControls {
		if len(strings.TrimSpace(control)) < 10 {
			problems = append(problems, "compensating controls must contain at least 10 characters")
		}
	}
	problems = append(problems, validateScope(exception.FindingType, exception.Scope)...)
	problems = append(problems, validateOwnership(policy, exception, changeAuthor)...)
	problems = append(problems, validateTimeline(policy, exception, now)...)
	if err := validateIssueURL(exception.RemediationIssue); err != nil {
		problems = append(problems, err.Error())
	}
	if exception.RenewalOf != "" {
		if !exceptionIDPattern.MatchString(exception.RenewalOf) || exception.RenewalOf == exception.ID {
			problems = append(problems, "renewal_of must identify a different valid exception id")
		}
		if len(strings.TrimSpace(exception.RenewalReason)) < 20 {
			problems = append(problems, "renewal_reason is required for a reviewed renewal")
		}
	} else if exception.RenewalReason != "" {
		problems = append(problems, "renewal_reason requires renewal_of")
	}
	return problems
}

func validateScope(findingType string, scope ExceptionScope) []string {
	var problems []string
	if len(strings.TrimSpace(scope.Component)) < 2 || broad(scope.Component) {
		problems = append(problems, "scope.component must identify one exact component")
	}
	if !contains(validComponentTypes, scope.ComponentType) {
		problems = append(problems, "scope.component_type is invalid")
	}
	if scope.Path != "" {
		if broad(scope.Path) || filepath.IsAbs(scope.Path) || strings.Contains(filepath.Clean(scope.Path), "..") {
			problems = append(problems, "scope.path must be an exact repository-relative path without traversal or globs")
		}
	}
	if oneOf(scope.ComponentType, "source", "workflow", "configuration", "kubernetes", "iac") && scope.Path == "" {
		problems = append(problems, "scope.path is required for source and configuration findings")
	}
	if !contains(validEnvironments, scope.Environment) {
		problems = append(problems, "scope.environment is invalid")
	}
	if !contains(validContexts, scope.Context) {
		problems = append(problems, "scope.context is invalid")
	}
	if err := validateVersionRange(scope.AffectedVersionRange); err != nil {
		problems = append(problems, err.Error())
	}
	if scope.ArtifactDigest != "" && !digestPattern.MatchString(scope.ArtifactDigest) {
		problems = append(problems, "scope.artifact_digest must be a lower-case sha256 digest")
	}
	if oneOf(findingType, "runtime_image", "development_image", "plugin_permission") && scope.ArtifactDigest == "" {
		problems = append(problems, "image and plugin exceptions require an exact artifact_digest")
	}
	return problems
}

func validateOwnership(policy Policy, exception Exception, changeAuthor string) []string {
	var problems []string
	for label, principal := range map[string]string{
		"owner":        exception.Ownership.Owner,
		"requested_by": exception.Ownership.RequestedBy,
		"approved_by":  exception.Ownership.ApprovedBy,
	} {
		if !principalPattern.MatchString(principal) {
			problems = append(problems, "ownership."+label+" must be a github:USER or team:ORG/TEAM principal")
		}
	}
	limit, ok := policy.ExceptionLimits[exception.RiskClass]
	if !ok {
		return problems
	}
	if !contains(limit.ApproverRoles, exception.Ownership.ApproverRole) {
		problems = append(problems, "ownership.approver_role is not authorized for the risk class")
	}
	if limit.IndependentApproval && exception.Ownership.RequestedBy == exception.Ownership.ApprovedBy {
		problems = append(problems, "high and critical exceptions require an approver independent of the requester")
	}
	if limit.IndependentApproval && changeAuthor != "" && normalizePrincipal(changeAuthor) == exception.Ownership.ApprovedBy {
		problems = append(problems, "high and critical exceptions cannot be approved by the change author")
	}
	return problems
}

func validateTimeline(policy Policy, exception Exception, now time.Time) []string {
	var problems []string
	approved, err := time.Parse(time.RFC3339, exception.Timeline.ApprovedAt)
	if err != nil {
		problems = append(problems, "timeline.approved_at must be RFC3339")
	}
	expires, err := time.Parse(time.RFC3339, exception.Timeline.ExpiresAt)
	if err != nil {
		problems = append(problems, "timeline.expires_at must be RFC3339")
	}
	lastReviewed, err := time.Parse(time.RFC3339, exception.Timeline.LastReviewedAt)
	if err != nil {
		problems = append(problems, "timeline.last_reviewed_at must be RFC3339")
	}
	nextReview, err := time.Parse(time.RFC3339, exception.Timeline.NextReviewAt)
	if err != nil {
		problems = append(problems, "timeline.next_review_at must be RFC3339")
	}
	if len(problems) > 0 {
		return problems
	}
	limit := policy.ExceptionLimits[exception.RiskClass]
	if approved.After(now.Add(5 * time.Minute)) {
		problems = append(problems, "timeline.approved_at must not be in the future")
	}
	if !expires.After(now) {
		problems = append(problems, "exception is expired")
	}
	if !expires.After(approved) {
		problems = append(problems, "timeline.expires_at must be after approved_at")
	}
	if expires.Sub(approved) > time.Duration(limit.MaxDurationHours)*time.Hour {
		problems = append(problems, "exception exceeds maximum duration for risk class")
	}
	if lastReviewed.Before(approved) || lastReviewed.After(now.Add(5*time.Minute)) {
		problems = append(problems, "timeline.last_reviewed_at must be between approval and now")
	}
	if !nextReview.After(now) {
		problems = append(problems, "exception review is overdue")
	}
	if !nextReview.After(lastReviewed) || nextReview.After(expires) {
		problems = append(problems, "timeline.next_review_at must be after last review and no later than expiry")
	}
	if nextReview.Sub(lastReviewed) > time.Duration(limit.ReviewIntervalHours)*time.Hour {
		problems = append(problems, "exception review interval exceeds policy for risk class")
	}
	return problems
}

func validateIssueURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("remediation_issue must be an absolute HTTPS URL")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("remediation_issue must not contain query parameters or fragments")
	}
	return nil
}

func normalizePrincipal(value string) string {
	value = strings.TrimPrefix(value, "@")
	if strings.HasPrefix(value, "github:") || strings.HasPrefix(value, "team:") {
		return value
	}
	return "github:" + value
}

func broad(value string) bool {
	return strings.ContainsAny(value, "*?[]{}") || strings.EqualFold(strings.TrimSpace(value), "all")
}

func oneOf(value string, allowed ...string) bool {
	return contains(allowed, value)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func duplicateValue(values []string) string {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return value
		}
		seen[value] = true
	}
	return ""
}

func joined(problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("security policy validation failed:\n- %s", strings.Join(problems, "\n- "))
}
