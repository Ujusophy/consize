package resource

import "time"

const (
	TypeKubernetesDeployment = "kubernetes.deployment"
	TypeAWSRDSInstance       = "aws.rds.instance"
	TypeStorageBucket        = "storage.bucket"
	TypeLLMApplication       = "llm.application"
)

const (
	ProviderKubernetes = "kubernetes"
	ProviderAWS        = "aws"
	ProviderGCP        = "gcp"
	ProviderAzure      = "azure"
	ProviderOpenAI     = "openai"
)

const (
	EnvProduction  = "production"
	EnvStaging     = "staging"
	EnvDevelopment = "development"
)

const (
	CriticalityLow    = "low"
	CriticalityMedium = "medium"
	CriticalityHigh   = "high"
)

// Resource is the common model for anything Consize can optimize.
// Provider-specific fields belong in Metadata and CurrentState so core logic
// stays portable across Kubernetes, cloud, storage, databases, and future systems.
type Resource struct {
	ID                 string            `json:"id"`
	Type               string            `json:"type"`
	Provider           string            `json:"provider"`
	ProviderResourceID string            `json:"provider_resource_id"`
	Name               string            `json:"name"`
	Environment        string            `json:"environment"`
	Owner              string            `json:"owner"`
	Region             string            `json:"region"`
	Account            string            `json:"account"`
	Criticality        string            `json:"criticality"`
	Labels             map[string]string `json:"labels"`
	Metadata           map[string]any    `json:"metadata"`
	CurrentState       map[string]any    `json:"current_state"`
	MonthlyCost        float64           `json:"monthly_cost"`
	SourcePluginID     string            `json:"source_plugin_id"`
	ObservedAt         time.Time         `json:"observed_at"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

func (r Resource) WithDefaults(now time.Time) Resource {
	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}
	if r.CurrentState == nil {
		r.CurrentState = map[string]any{}
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	return r
}
