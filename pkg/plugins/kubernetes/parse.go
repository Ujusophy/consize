package kubernetes

import (
	"errors"
	"fmt"
)

func patchDiffFromParameters(params map[string]any) (PatchDiff, error) {
	if params == nil {
		return PatchDiff{}, errors.New("kubernetes action requires parameters")
	}
	return patchDiffFromAny(params["patch"])
}

func patchDiffFromAny(v any) (PatchDiff, error) {
	switch t := v.(type) {
	case PatchDiff:
		return validatePatchDiff(t)
	case map[string]any:
		return validatePatchDiff(PatchDiff{
			Resource:      stringValue(t["resource"]),
			CurrentReq:    int64Value(t["current_request"]),
			ProposedReq:   int64Value(t["proposed_request"]),
			CurrentLimit:  int64Value(t["current_limit"]),
			ProposedLimit: int64Value(t["proposed_limit"]),
		})
	default:
		return PatchDiff{}, fmt.Errorf("patch must be an object, got %T", v)
	}
}

func validatePatchDiff(diff PatchDiff) (PatchDiff, error) {
	if diff.Resource != "cpu" && diff.Resource != "memory" {
		return PatchDiff{}, errors.New("patch.resource must be cpu or memory")
	}
	if diff.ProposedReq <= 0 {
		return PatchDiff{}, errors.New("patch.proposed_request must be greater than zero")
	}
	if diff.CurrentReq < 0 || diff.CurrentLimit < 0 || diff.ProposedLimit < 0 {
		return PatchDiff{}, errors.New("patch values cannot be negative")
	}
	if diff.CurrentLimit > 0 && diff.ProposedLimit == 0 {
		return PatchDiff{}, errors.New("patch cannot remove an existing limit")
	}
	return diff, nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func int64Value(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case jsonNumber:
		n, _ := t.Int64()
		return n
	default:
		return 0
	}
}

type jsonNumber interface {
	Int64() (int64, error)
}
