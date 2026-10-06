package store

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func contractFixture(t *testing.T) (*Memory, Recommendation) {
	t.Helper()
	st := NewMemory()
	res := completeResource("resource-1")
	if _, err := st.UpsertResource(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	rec, err := st.CreateRecommendation(context.Background(), Recommendation{
		ResourceID: res.ID, PluginID: "kubernetes-action", ActionType: "k8s.patch_resources",
		AlgorithmID: "headroom", AlgorithmVersion: "2.1.0", EvidenceRefs: []string{"metrics:prometheus:window-1"},
		Current: map[string]any{"request": 100}, Proposed: map[string]any{"request": 75},
	})
	if err != nil {
		t.Fatal(err)
	}
	return st, rec
}

func TestRecommendationTransitionTable(t *testing.T) {
	for _, tc := range []struct {
		from RecommendationStatus
		to   RecommendationStatus
		want bool
	}{
		{RecommendationPending, RecommendationPlanned, true},
		{RecommendationPending, RecommendationVerified, false},
		{RecommendationExecuting, RecommendationVerified, true},
		{RecommendationVerified, RecommendationPending, false},
	} {
		if got := ValidRecommendationTransition(tc.from, tc.to); got != tc.want {
			t.Fatalf("transition %s -> %s = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestActionTransitionTable(t *testing.T) {
	for _, tc := range []struct {
		from ActionStatus
		to   ActionStatus
		want bool
	}{
		{ActionRequested, ActionPlanned, true},
		{ActionRequested, ActionSucceeded, false},
		{ActionExecuting, ActionVerifying, true},
		{ActionSucceeded, ActionExecuting, false},
	} {
		if got := ValidActionTransition(tc.from, tc.to); got != tc.want {
			t.Fatalf("transition %s -> %s = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestStoreRejectsIllegalTransitions(t *testing.T) {
	st, rec := contractFixture(t)
	if err := st.TransitionRecommendation(context.Background(), rec.ID, RecommendationVerified, "skipped execution"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("recommendation transition error = %v", err)
	}
	action, err := st.CreateAction(context.Background(), Action{RecommendationID: rec.ID, ResourceID: rec.ResourceID, RemediationPath: "direct_apply", PluginID: rec.PluginID, ActionType: rec.ActionType, Mode: "approved", IdempotencyKey: "illegal-transition", RequestedBy: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.TransitionAction(context.Background(), action.ID, ActionSucceeded, nil); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("action transition error = %v", err)
	}
}

func TestActionTransitionCannotRewriteImmutableIdentity(t *testing.T) {
	st, rec := contractFixture(t)
	action, err := st.CreateAction(context.Background(), Action{RecommendationID: rec.ID, ResourceID: rec.ResourceID, RemediationPath: "direct_apply", PluginID: rec.PluginID, ActionType: rec.ActionType, Mode: "approved", IdempotencyKey: "immutable", RequestedBy: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.TransitionAction(context.Background(), action.ID, ActionApproved, func(candidate *Action) error {
		candidate.ResourceID = "another-resource"
		return nil
	}); err == nil {
		t.Fatal("immutable action identity was rewritten")
	}
}

func TestCreateActionIsIdempotentAndTraceable(t *testing.T) {
	st, rec := contractFixture(t)
	input := Action{RecommendationID: rec.ID, ResourceID: rec.ResourceID, RemediationPath: "direct_apply", PluginID: rec.PluginID, ActionType: rec.ActionType, Mode: "approved", IdempotencyKey: "request-123", RequestedBy: "operator"}
	first, err := st.CreateAction(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.CreateAction(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("duplicate action IDs: %d and %d", first.ID, second.ID)
	}
	if first.RecommendationID != rec.ID || first.ResourceID != rec.ResourceID {
		t.Fatal("action traceability was not preserved")
	}
}

func TestIdempotencyKeyCannotBeReusedForDifferentRequest(t *testing.T) {
	st, rec := contractFixture(t)
	first := Action{RecommendationID: rec.ID, ResourceID: rec.ResourceID, RemediationPath: "direct_apply", PluginID: rec.PluginID, ActionType: rec.ActionType, Mode: "dry_run", IdempotencyKey: "shared-key", RequestedBy: "operator"}
	if _, err := st.CreateAction(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	first.Mode = "approved"
	if _, err := st.CreateAction(context.Background(), first); err == nil {
		t.Fatal("idempotency key was rebound to a different request")
	}
}

func TestActionWithoutRecommendationIsRejected(t *testing.T) {
	st := NewMemory()
	_, err := st.CreateAction(context.Background(), Action{RecommendationID: 42, ResourceID: "missing", RemediationPath: "direct_apply", PluginID: "plugin", ActionType: "resize", Mode: "approved", IdempotencyKey: "key", RequestedBy: "operator"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want not found", err)
	}
}

func TestExpiredAndSupersededRecommendationsCannotExecute(t *testing.T) {
	for _, status := range []RecommendationStatus{RecommendationExpired, RecommendationSuperseded} {
		t.Run(string(status), func(t *testing.T) {
			st, rec := contractFixture(t)
			if status == RecommendationSuperseded {
				replacement, err := st.CreateRecommendation(context.Background(), Recommendation{ResourceID: rec.ResourceID, PluginID: rec.PluginID, ActionType: rec.ActionType, AlgorithmID: "headroom", EvidenceRefs: []string{"metrics:new"}})
				if err != nil {
					t.Fatal(err)
				}
				if err := st.SupersedeRecommendation(context.Background(), rec.ID, replacement.ID, "new evidence"); err != nil {
					t.Fatal(err)
				}
			} else if err := st.TransitionRecommendation(context.Background(), rec.ID, RecommendationExpired, "evidence window elapsed"); err != nil {
				t.Fatal(err)
			}
			_, err := st.CreateAction(context.Background(), Action{RecommendationID: rec.ID, ResourceID: rec.ResourceID, RemediationPath: "direct_apply", PluginID: rec.PluginID, ActionType: rec.ActionType, Mode: "approved", IdempotencyKey: "blocked-" + string(status), RequestedBy: "operator"})
			if !errors.Is(err, ErrRecommendationGone) {
				t.Fatalf("error = %v, want recommendation gone", err)
			}
		})
	}
}

func TestRecommendationExpiryTimeBlocksAction(t *testing.T) {
	st := NewMemory()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	st.now = func() time.Time { return now }
	res := completeResource("resource-1")
	if _, err := st.UpsertResource(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	rec, err := st.CreateRecommendation(context.Background(), Recommendation{ResourceID: res.ID, PluginID: "plugin", ActionType: "resize", AlgorithmID: "algorithm", EvidenceRefs: []string{"evidence:1"}, ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	_, err = st.CreateAction(context.Background(), Action{RecommendationID: rec.ID, ResourceID: rec.ResourceID, RemediationPath: "direct_apply", PluginID: rec.PluginID, ActionType: rec.ActionType, Mode: "approved", IdempotencyKey: "expired-time", RequestedBy: "operator"})
	if !errors.Is(err, ErrRecommendationGone) {
		t.Fatalf("error = %v, want recommendation gone", err)
	}
}

func TestRecommendationAndActionPersistRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, err := OpenDurable(path)
	if err != nil {
		t.Fatal(err)
	}
	res := completeResource("resource-1")
	if _, err = st.UpsertResource(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	rec, err := st.CreateRecommendation(context.Background(), Recommendation{ResourceID: res.ID, PluginID: "plugin", ActionType: "resize", AlgorithmID: "algorithm", AlgorithmVersion: "1.2.3", EvidenceRefs: []string{"evidence:1"}, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	action, err := st.CreateAction(context.Background(), Action{RecommendationID: rec.ID, ResourceID: res.ID, RemediationPath: "direct_apply", PluginID: "plugin", ActionType: "resize", Mode: "dry_run", IdempotencyKey: "round-trip", RequestedBy: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = OpenDurable(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	gotRec, err := st.GetRecommendation(context.Background(), rec.ID)
	if err != nil || gotRec.AlgorithmVersion != "1.2.3" || gotRec.EvidenceRefs[0] != "evidence:1" {
		t.Fatalf("recommendation round trip: %#v, %v", gotRec, err)
	}
	gotAction, err := st.GetAction(context.Background(), action.ID)
	if err != nil || gotAction.IdempotencyKey != "round-trip" || gotAction.RecommendationID != rec.ID {
		t.Fatalf("action round trip: %#v, %v", gotAction, err)
	}
	if _, err := json.Marshal(gotAction); err != nil {
		t.Fatal(err)
	}
}

func TestRecommendationRequiresEvidenceAndAlgorithm(t *testing.T) {
	st := NewMemory()
	res := completeResource("resource-1")
	if _, err := st.UpsertResource(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateRecommendation(context.Background(), Recommendation{ResourceID: res.ID, PluginID: "plugin", ActionType: "resize"}); err == nil {
		t.Fatal("recommendation without evidence and algorithm was accepted")
	}
}
