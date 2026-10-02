package resource

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func validResource() Resource {
	return Resource{
		Type: TypeKubernetesDeployment, Provider: ProviderKubernetes,
		ProviderResourceID: "payments/checkout-api", Name: "checkout-api",
		Environment: EnvProduction, Owner: "payments", Region: "us-east-1",
		Account: "cluster-a", Criticality: CriticalityHigh,
		Metadata:     map[string]any{"namespace": "payments", "generation": float64(7)},
		CurrentState: map[string]any{"memory_request_bytes": float64(536870912)},
	}
}

func TestCanonicalIDUsesOnlyImmutableIdentity(t *testing.T) {
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	first, err := validResource().Normalize(now)
	if err != nil {
		t.Fatal(err)
	}
	changed := validResource()
	changed.Owner = "new-owner"
	changed.Environment = EnvStaging
	changed.Criticality = CriticalityLow
	changed.Metadata["generation"] = float64(8)
	changed.CurrentState["memory_request_bytes"] = float64(268435456)
	second, err := changed.Normalize(now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("mutable fields changed identity: %q != %q", first.ID, second.ID)
	}
	want := "resource:v1:kubernetes:cluster-a:us-east-1:kubernetes.deployment:payments%2Fcheckout-api"
	if first.ID != want {
		t.Fatalf("id = %q, want %q", first.ID, want)
	}
}

func TestIdentitySeparatesProvidersClustersAndLocations(t *testing.T) {
	base := validResource().Identity()
	ids := map[string]bool{}
	for _, identity := range []Identity{
		base,
		{Provider: ProviderAWS, Account: base.Account, Location: base.Location, Type: base.Type, ProviderResourceID: base.ProviderResourceID},
		{Provider: base.Provider, Account: "cluster-b", Location: base.Location, Type: base.Type, ProviderResourceID: base.ProviderResourceID},
		{Provider: base.Provider, Account: base.Account, Location: "eu-west-1", Type: base.Type, ProviderResourceID: base.ProviderResourceID},
	} {
		id, err := BuildID(identity)
		if err != nil {
			t.Fatal(err)
		}
		if ids[id] {
			t.Fatalf("identity collision for %q", id)
		}
		ids[id] = true
	}
}

func TestNormalizeRejectsInvalidIdentity(t *testing.T) {
	res := validResource()
	res.Account = ""
	if _, err := res.Normalize(time.Now()); !errors.Is(err, ErrInvalidResource) {
		t.Fatalf("error = %v", err)
	}
	res = validResource()
	res.ProviderResourceID = "bad\nidentity"
	if _, err := res.Normalize(time.Now()); !errors.Is(err, ErrInvalidResource) {
		t.Fatalf("error = %v", err)
	}
}

func TestSupportStatusIsDerivedAndUnsupportedIsExplicit(t *testing.T) {
	res := validResource()
	res.Type = "custom.database"
	res.SupportStatus = SupportFull
	normalized, err := res.Normalize(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if normalized.SupportStatus != SupportUnsupported {
		t.Fatalf("support status = %q", normalized.SupportStatus)
	}
}

func TestProviderMetadataJSONRoundTrip(t *testing.T) {
	normalized, err := validResource().Normalize(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Resource
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatal(err)
	}
	if decoded.Metadata["namespace"] != "payments" || decoded.Metadata["generation"] != float64(7) {
		t.Fatalf("metadata changed: %#v", decoded.Metadata)
	}
}

func TestLifecycleTransitions(t *testing.T) {
	if !ValidLifecycleTransition(LifecycleActive, LifecycleStale) || !ValidLifecycleTransition(LifecycleStale, LifecycleDeleted) || !ValidLifecycleTransition(LifecycleDeleted, LifecycleActive) {
		t.Fatal("supported lifecycle transition rejected")
	}
	if ValidLifecycleTransition(LifecycleDeleted, LifecycleStale) {
		t.Fatal("deleted resource transitioned directly to stale")
	}
}
