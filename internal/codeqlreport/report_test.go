package codeqlreport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/consize-oss/consize/internal/securitypolicy"
)

const fixture = `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"CodeQL","rules":[{"id":"go/synthetic","properties":{"security-severity":"9.1","precision":"high"}}]}},"invocations":[{"executionSuccessful":true}],"results":[{"ruleId":"go/synthetic","message":{"text":"DO NOT RETAIN source secret"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"internal/example.go"},"region":{"startLine":5}}}]}]}]}`

func TestNormalizeAndFailClosed(t *testing.T) {
	target := securitypolicy.ReportTarget{Component: "consize-source", Environment: "ci", Context: "build", Version: "synthetic-revision"}
	report, err := Normalize([]byte(fixture), target, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || report.Findings[0].Severity != "critical" || report.Findings[0].Confidence != "high" || report.Findings[0].Reachability != "unknown" {
		t.Fatalf("incorrect finding: %+v", report)
	}
	b, _ := json.Marshal(report)
	if strings.Contains(string(b), "DO NOT RETAIN") {
		t.Fatal("source text leaked")
	}
	for _, invalid := range []string{"{", `{"version":"2.1.0","runs":[]}`, strings.Replace(fixture, `"executionSuccessful":true`, `"executionSuccessful":false`, 1), strings.Replace(fixture, `go/synthetic","message`, `unknown/rule","message`, 1), strings.Replace(fixture, `internal/example.go`, `../outside`, 1)} {
		if _, err := Normalize([]byte(invalid), target, time.Now()); err == nil {
			t.Fatalf("accepted invalid input %s", invalid)
		}
	}
	clean := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"CodeQL","rules":[]}},"results":[]}]}`
	report, err = Normalize([]byte(clean), target, time.Now())
	if err != nil || len(report.Findings) != 0 {
		t.Fatal("clean result rejected")
	}
}

func TestCodeQLPackRuleReferences(t *testing.T) {
	input := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"CodeQL"},"extensions":[{"rules":[{"id":"js/example","properties":{"security-severity":"8","precision":"high"}}]}]},"results":[{"rule":{"index":0,"toolComponent":{"index":0}},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"ui/src/example.ts"},"region":{"startLine":1}}}]}]}]}`
	report, err := Normalize([]byte(input), securitypolicy.ReportTarget{Component: "consize-source", Environment: "ci", Context: "build", Version: "test-revision"}, time.Now())
	if err != nil || len(report.Findings) != 1 || report.Findings[0].RuleID != "js/example" {
		t.Fatalf("pack rule reference: %+v %v", report, err)
	}
}
