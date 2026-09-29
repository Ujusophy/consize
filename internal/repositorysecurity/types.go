package repositorysecurity

import "time"

const (
	SchemaVersion   = 1
	ScannerID       = "repository-security"
	DefaultMaxBytes = int64(1 << 20)
	HardMaxBytes    = int64(25 << 20)
)

type TrackedFile struct {
	Path       string
	Mode       string
	GitHash    string
	Size       int64
	Content    []byte
	SHA256     string
	ContentSet bool
}

type DeclarationRegistry struct {
	Schema        string        `json:"$schema"`
	SchemaVersion int           `json:"schema_version"`
	Declarations  []Declaration `json:"declarations"`
}

type Declaration struct {
	ID             string `json:"id"`
	RuleID         string `json:"rule_id"`
	Path           string `json:"path"`
	SHA256         string `json:"sha256"`
	Classification string `json:"classification"`
	Reason         string `json:"reason"`
	Owner          string `json:"owner"`
	ApprovedBy     string `json:"approved_by"`
	ReviewedAt     string `json:"reviewed_at"`
	ExpiresAt      string `json:"expires_at"`
}

type Candidate struct {
	RuleID           string
	Path             string
	FindingType      string
	Severity         string
	ComponentType    string
	Summary          string
	ExceptionAllowed bool
}

type ScanOptions struct {
	Environment string
	Context     string
	Revision    string
	GeneratedAt time.Time
	MaxBytes    int64
}

type Result struct {
	Candidates         []Candidate
	Suppressed         []SuppressedFinding
	UnusedDeclarations []Declaration
}

type SuppressedFinding struct {
	Candidate     Candidate
	DeclarationID string
}
