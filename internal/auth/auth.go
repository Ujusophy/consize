package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	RoleViewer   = "viewer"
	RoleOperator = "operator"
	RoleAdmin    = "admin"
)

type TokenConfig struct {
	Subject  string `json:"subject"`
	Role     string `json:"role"`
	TokenEnv string `json:"token_env"`
}

type Config struct {
	Enabled bool          `json:"enabled"`
	Tokens  []TokenConfig `json:"tokens"`
}

type Identity struct {
	Subject string `json:"subject"`
	Role    string `json:"role"`
}

type credential struct {
	identity Identity
	hash     [sha256.Size]byte
}

type Authorizer struct {
	enabled     bool
	credentials []credential
}

type contextKey struct{}

func New(cfg Config) (*Authorizer, error) {
	a := &Authorizer{enabled: cfg.Enabled}
	if !cfg.Enabled {
		return a, nil
	}
	if len(cfg.Tokens) == 0 {
		return nil, errors.New("authentication is enabled but no tokens are configured")
	}
	seen := map[string]bool{}
	for _, token := range cfg.Tokens {
		if strings.TrimSpace(token.Subject) == "" || !validRole(token.Role) || strings.TrimSpace(token.TokenEnv) == "" {
			return nil, errors.New("auth tokens require subject, valid role, and token_env")
		}
		if seen[token.Subject] {
			return nil, fmt.Errorf("duplicate auth subject %q", token.Subject)
		}
		value, ok := os.LookupEnv(token.TokenEnv)
		if !ok || len(value) < 24 {
			return nil, fmt.Errorf("auth token environment variable %s is missing or too short", token.TokenEnv)
		}
		seen[token.Subject] = true
		a.credentials = append(a.credentials, credential{identity: Identity{Subject: token.Subject, Role: token.Role}, hash: sha256.Sum256([]byte(value))})
	}
	return a, nil
}

func (a *Authorizer) Authenticate(r *http.Request) (Identity, error) {
	if !a.enabled {
		return Identity{Subject: "local-operator", Role: RoleAdmin}, nil
	}
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")) == "" {
		return Identity{}, errors.New("bearer token is required")
	}
	presented := sha256.Sum256([]byte(strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))))
	for _, candidate := range a.credentials {
		if subtle.ConstantTimeCompare(presented[:], candidate.hash[:]) == 1 {
			return candidate.identity, nil
		}
	}
	return Identity{}, errors.New("invalid bearer token")
}

func (a *Authorizer) Authorize(r *http.Request, requiredRole string) (*http.Request, Identity, error) {
	identity, err := a.Authenticate(r)
	if err != nil {
		return r, Identity{}, err
	}
	if roleRank(identity.Role) < roleRank(requiredRole) {
		return r, identity, fmt.Errorf("role %s is required", requiredRole)
	}
	ctx := context.WithValue(r.Context(), contextKey{}, identity)
	return r.WithContext(ctx), identity, nil
}

func FromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(contextKey{}).(Identity)
	return identity, ok
}

func validRole(role string) bool { return roleRank(role) > 0 }

func roleRank(role string) int {
	switch role {
	case RoleViewer:
		return 1
	case RoleOperator:
		return 2
	case RoleAdmin:
		return 3
	default:
		return 0
	}
}
