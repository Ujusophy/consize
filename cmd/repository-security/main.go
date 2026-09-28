package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/consize-oss/consize/internal/repositorysecurity"
)

func main() {
	root := flag.String("root", ".", "repository root")
	declarationsPath := flag.String("declarations", ".github/security/repository-allowlist.json", "reviewed repository declarations")
	output := flag.String("output", "repository-security-report.json", "normalized report output")
	environment := flag.String("environment", "ci", "scan environment")
	contextName := flag.String("context", "pull_request", "scan context")
	maximum := flag.Int64("max-bytes", repositorysecurity.DefaultMaxBytes, "maximum tracked file size")
	flag.Parse()

	now := time.Now().UTC()
	declarations, err := repositorysecurity.LoadDeclarations(*declarationsPath, now)
	if err != nil {
		fatal(err)
	}
	report, result, err := repositorysecurity.Scan(context.Background(), *root, declarations, repositorysecurity.ScanOptions{
		Environment: *environment, Context: *contextName, GeneratedAt: now, MaxBytes: *maximum,
	})
	if err != nil {
		fatal(err)
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, append(encoded, '\n'), 0o600); err != nil {
		fatal(err)
	}
	for _, finding := range result.Candidates {
		fmt.Printf("%s: %s: %s\n", finding.Path, finding.RuleID, finding.Summary)
	}
	fmt.Printf("repository scan completed: %d finding(s), %d reviewed declaration(s); report: %s\n", len(result.Candidates), len(result.Suppressed), *output)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "repository security scan failed:", err)
	os.Exit(2)
}
