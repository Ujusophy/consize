package gitleaksreport

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeRedactedFinding(t *testing.T) {
	input := `[{
      "RuleID":"generic-api-key","Description":"key","StartLine":7,"EndLine":7,
      "StartColumn":1,"EndColumn":10,"Match":"REDACTED","Secret":"REDACTED",
      "File":"config/test.env","SymlinkFile":"","Commit":"0123456789abcdef",
      "Entropy":4.2,"Author":"","Email":"","Date":"","Message":"","Tags":[],
      "Fingerprint":"0123456789abcdef:config/test.env:generic-api-key:7"
    }]`
	report, err := Normalize(strings.NewReader(input), Options{Context: "pull_request", Version: "0123456789abcdef", ScanID: "gitleaks-test", Now: time.Unix(0, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || !report.Findings[0].Redacted {
		t.Fatalf("unexpected report: %#v", report)
	}
	if strings.Contains(report.Findings[0].Summary, "REDACTED") {
		t.Fatal("summary should contain metadata only")
	}
}

func TestNormalizeRejectsUnredactedFinding(t *testing.T) {
	value := "sen" + "sitive"
	input := `[{"RuleID":"x","File":"x.env","Secret":"` + value + `","Match":"` + value + `"}]`
	_, err := Normalize(strings.NewReader(input), Options{Context: "pull_request", Version: "v", ScanID: "gitleaks-test", Now: time.Now()})
	if err == nil {
		t.Fatal("expected unredacted report to fail")
	}
}

func TestNormalizeRejectsUnknownFields(t *testing.T) {
	_, err := Normalize(strings.NewReader(`[{"RuleID":"x","File":"x","Unexpected":true}]`), Options{})
	if err == nil {
		t.Fatal("expected unknown scanner fields to fail")
	}
}
