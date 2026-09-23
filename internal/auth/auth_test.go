package auth

import (
	"net/http/httptest"
	"testing"
)

func TestAuthorizerUsesEnvironmentTokenAndRole(t *testing.T) {
	t.Setenv("CONSIZE_TEST_OPERATOR_TOKEN", "this-is-a-long-test-token-value")
	a, err := New(Config{Enabled: true, Tokens: []TokenConfig{{Subject: "operator@example.com", Role: RoleOperator, TokenEnv: "CONSIZE_TEST_OPERATOR_TOKEN"}}})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/api/recommendations/1/execute", nil)
	r.Header.Set("Authorization", "Bearer this-is-a-long-test-token-value")
	authorized, identity, err := a.Authorize(r, RoleOperator)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Subject != "operator@example.com" {
		t.Fatalf("identity = %#v", identity)
	}
	if fromContext, ok := FromContext(authorized.Context()); !ok || fromContext != identity {
		t.Fatalf("context identity = %#v, %v", fromContext, ok)
	}
	if _, _, err := a.Authorize(r, RoleAdmin); err == nil {
		t.Fatal("operator was authorized as admin")
	}
}

func TestAuthorizerRejectsMissingCredential(t *testing.T) {
	t.Setenv("CONSIZE_TEST_VIEWER_TOKEN", "this-is-a-long-viewer-token")
	a, err := New(Config{Enabled: true, Tokens: []TokenConfig{{Subject: "viewer@example.com", Role: RoleViewer, TokenEnv: "CONSIZE_TEST_VIEWER_TOKEN"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Authenticate(httptest.NewRequest("GET", "/api/dashboard", nil)); err == nil {
		t.Fatal("missing token was accepted")
	}
}
