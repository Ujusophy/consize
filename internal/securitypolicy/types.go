package securitypolicy

import "time"

const (
	SchemaVersion = 1

	DecisionBlock    = "block"
	DecisionExcepted = "excepted"
	DecisionTriage   = "triage"

	GatePassed     = "passed"
	GateBlocked    = "blocked"
	GateIncomplete = "incomplete"
)

type Policy struct {
	Schema          string                    `json:"$schema"`
	SchemaVersion   int                       `json:"schema_version"`
	PolicyVersion   string                    `json:"policy_version"`
	ExceptionLimits map[string]ExceptionLimit `json:"exception_limits"`
	ServiceLevels   map[string]ServiceLevel   `json:"service_levels"`
	FindingRules    []FindingRule             `json:"finding_rules"`
}

type ExceptionLimit struct {
	MaxDurationHours    int      `json:"max_duration_hours"`
	ReviewIntervalHours int      `json:"review_interval_hours"`
	IndependentApproval bool     `json:"independent_approval"`
	ApproverRoles       []string `json:"approver_roles"`
}

type ServiceLevel struct {
	AcknowledgementHours     int    `json:"acknowledgement_hours"`
	RemediationDecisionHours int    `json:"remediation_or_decision_hours"`
	ReleaseImpact            string `json:"release_impact"`
}

type FindingRule struct {
	FindingType                  string   `json:"finding_type"`
	Description                  string   `json:"description"`
	BlockMode                    string   `json:"block_mode"`
	BlockSeverities              []string `json:"block_severities,omitempty"`
	BlockedReachability          []string `json:"blocked_reachability,omitempty"`
	MinimumConfidence            string   `json:"minimum_confidence,omitempty"`
	MandatoryOnly                bool     `json:"mandatory_only,omitempty"`
	ExceptionMode                string   `json:"exception_mode"`
	RequiredExceptionEnvironment string   `json:"required_exception_environment,omitempty"`
	ReleaseImpact                string   `json:"release_impact"`
	Owner                        string   `json:"owner"`
}

type ExceptionRegistry struct {
	Schema        string      `json:"$schema"`
	SchemaVersion int         `json:"schema_version"`
	PolicyVersion string      `json:"policy_version"`
	Exceptions    []Exception `json:"exceptions"`
}

type Exception struct {
	ID                    string             `json:"id"`
	Scanner               string             `json:"scanner"`
	FindingType           string             `json:"finding_type"`
	RuleID                string             `json:"rule_id"`
	Fingerprint           string             `json:"fingerprint"`
	RiskClass             string             `json:"risk_class"`
	Disposition           string             `json:"disposition"`
	Scope                 ExceptionScope     `json:"scope"`
	Reason                string             `json:"reason"`
	FalsePositiveEvidence string             `json:"false_positive_evidence,omitempty"`
	CompensatingControls  []string           `json:"compensating_controls"`
	Ownership             ExceptionOwnership `json:"ownership"`
	Timeline              ExceptionTimeline  `json:"timeline"`
	RemediationIssue      string             `json:"remediation_issue"`
	RenewalOf             string             `json:"renewal_of,omitempty"`
	RenewalReason         string             `json:"renewal_reason,omitempty"`
}

type ExceptionScope struct {
	Component            string `json:"component"`
	ComponentType        string `json:"component_type"`
	Path                 string `json:"path,omitempty"`
	ArtifactDigest       string `json:"artifact_digest,omitempty"`
	Environment          string `json:"environment"`
	Context              string `json:"context"`
	AffectedVersionRange string `json:"affected_version_range"`
}

type ExceptionOwnership struct {
	Owner        string `json:"owner"`
	RequestedBy  string `json:"requested_by"`
	ApprovedBy   string `json:"approved_by"`
	ApproverRole string `json:"approver_role"`
}

type ExceptionTimeline struct {
	ApprovedAt     string `json:"approved_at"`
	ExpiresAt      string `json:"expires_at"`
	LastReviewedAt string `json:"last_reviewed_at"`
	NextReviewAt   string `json:"next_review_at"`
}

type FindingReport struct {
	Schema        string       `json:"$schema"`
	SchemaVersion int          `json:"schema_version"`
	Scanner       string       `json:"scanner"`
	ScanID        string       `json:"scan_id"`
	GeneratedAt   string       `json:"generated_at"`
	Status        string       `json:"status"`
	Attempts      int          `json:"attempts"`
	Target        ReportTarget `json:"target"`
	Findings      []Finding    `json:"findings"`
}

type ReportTarget struct {
	Component      string `json:"component"`
	ArtifactDigest string `json:"artifact_digest,omitempty"`
	Environment    string `json:"environment"`
	Context        string `json:"context"`
	Version        string `json:"version"`
}

type Finding struct {
	Fingerprint      string `json:"fingerprint"`
	RuleID           string `json:"rule_id"`
	FindingType      string `json:"finding_type"`
	Severity         string `json:"severity"`
	Confidence       string `json:"confidence"`
	Reachability     string `json:"reachability"`
	Mandatory        bool   `json:"mandatory"`
	ExceptionAllowed bool   `json:"exception_allowed"`
	LiveCredential   bool   `json:"live_credential"`
	Component        string `json:"component"`
	ComponentType    string `json:"component_type"`
	Path             string `json:"path,omitempty"`
	ArtifactDigest   string `json:"artifact_digest,omitempty"`
	Environment      string `json:"environment"`
	Context          string `json:"context"`
	Version          string `json:"version"`
	Summary          string `json:"summary"`
	Redacted         bool   `json:"redacted"`
	Owner            string `json:"owner,omitempty"`
	RemediationIssue string `json:"remediation_issue,omitempty"`
}

type Evaluation struct {
	SchemaVersion int        `json:"schema_version"`
	PolicyVersion string     `json:"policy_version"`
	ScanID        string     `json:"scan_id"`
	Scanner       string     `json:"scanner"`
	Gate          string     `json:"gate"`
	EvaluatedAt   time.Time  `json:"evaluated_at"`
	Counts        Counts     `json:"counts"`
	Decisions     []Decision `json:"decisions"`
}

type Counts struct {
	Blocked    int `json:"blocked"`
	Excepted   int `json:"excepted"`
	Triage     int `json:"triage"`
	Incomplete int `json:"incomplete"`
}

type Decision struct {
	Fingerprint      string `json:"fingerprint,omitempty"`
	RuleID           string `json:"rule_id,omitempty"`
	FindingType      string `json:"finding_type"`
	Severity         string `json:"severity,omitempty"`
	Decision         string `json:"decision"`
	Reason           string `json:"reason"`
	ExceptionID      string `json:"exception_id,omitempty"`
	Owner            string `json:"owner"`
	RemediationIssue string `json:"remediation_issue,omitempty"`
}

func (p Policy) rule(findingType string) (FindingRule, bool) {
	for _, rule := range p.FindingRules {
		if rule.FindingType == findingType {
			return rule, true
		}
	}
	return FindingRule{}, false
}
