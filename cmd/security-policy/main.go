package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/consize-oss/consize/internal/securitypolicy"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "validate":
		err = validate(os.Args[2:])
	case "evaluate":
		err = evaluate(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		var blocked *gateBlockedError
		if errors.As(err, &blocked) {
			os.Exit(1)
		}
		os.Exit(2)
	}
}

func validate(arguments []string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	policyPath := flags.String("policy", ".github/security/policy.json", "path to the security policy")
	exceptionsPath := flags.String("exceptions", ".github/security/exceptions.json", "path to the exception registry")
	baselinePath := flags.String("baseline-exceptions", "", "optional base-revision exception registry")
	at := flags.String("at", "", "RFC3339 evaluation time; defaults to now")
	changeAuthor := flags.String("change-author", "", "GitHub login of the change author")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	now, err := evaluationTime(*at)
	if err != nil {
		return err
	}
	policy, err := securitypolicy.LoadPolicy(*policyPath)
	if err != nil {
		return err
	}
	registry, err := securitypolicy.LoadExceptions(*exceptionsPath)
	if err != nil {
		return err
	}
	validationAuthor := *changeAuthor
	if *baselinePath != "" {
		validationAuthor = ""
	}
	if err := securitypolicy.ValidateExceptions(policy, registry, now, validationAuthor); err != nil {
		return err
	}
	if *baselinePath != "" {
		baseline, err := securitypolicy.LoadExceptions(*baselinePath)
		if err != nil {
			return err
		}
		if err := securitypolicy.ValidateExceptionChanges(baseline, registry, *changeAuthor); err != nil {
			return err
		}
	}
	fmt.Printf("security policy %s and %d exception(s) are valid at %s\n", policy.PolicyVersion, len(registry.Exceptions), now.UTC().Format(time.RFC3339))
	return nil
}

func evaluate(arguments []string) error {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	policyPath := flags.String("policy", ".github/security/policy.json", "path to the security policy")
	exceptionsPath := flags.String("exceptions", ".github/security/exceptions.json", "path to the exception registry")
	reportPath := flags.String("report", "", "path to a normalized scanner report")
	outputPath := flags.String("output", "", "optional path for the decision report")
	at := flags.String("at", "", "RFC3339 evaluation time; defaults to now")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *reportPath == "" {
		return fmt.Errorf("--report is required")
	}
	now, err := evaluationTime(*at)
	if err != nil {
		return err
	}
	policy, err := securitypolicy.LoadPolicy(*policyPath)
	if err != nil {
		return err
	}
	registry, err := securitypolicy.LoadExceptions(*exceptionsPath)
	if err != nil {
		return err
	}
	report, err := securitypolicy.LoadReport(*reportPath)
	if err != nil {
		return err
	}
	result, err := securitypolicy.Evaluate(policy, registry, report, now)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if *outputPath == "" {
		if _, err := os.Stdout.Write(encoded); err != nil {
			return err
		}
	} else if err := os.WriteFile(*outputPath, encoded, 0o600); err != nil {
		return err
	}
	if result.Gate != securitypolicy.GatePassed {
		return &gateBlockedError{gate: result.Gate}
	}
	return nil
}

func evaluationTime(value string) (time.Time, error) {
	if value == "" {
		return time.Now().UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --at value: %w", err)
	}
	return parsed.UTC(), nil
}

type gateBlockedError struct{ gate string }

func (e *gateBlockedError) Error() string { return "security gate is " + e.gate }

func usage() {
	fmt.Fprintln(os.Stderr, "usage: security-policy <validate|evaluate> [flags]")
}
