package securitypolicy

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func LoadPolicy(path string) (Policy, error) {
	var value Policy
	if err := decodeStrict(path, &value); err != nil {
		return Policy{}, fmt.Errorf("load policy: %w", err)
	}
	return value, nil
}

func LoadExceptions(path string) (ExceptionRegistry, error) {
	var value ExceptionRegistry
	if err := decodeStrict(path, &value); err != nil {
		return ExceptionRegistry{}, fmt.Errorf("load exceptions: %w", err)
	}
	return value, nil
}

func LoadReport(path string) (FindingReport, error) {
	var value FindingReport
	if err := decodeStrict(path, &value); err != nil {
		return FindingReport{}, fmt.Errorf("load report: %w", err)
	}
	return value, nil
}

func decodeStrict(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("file contains more than one JSON value")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}
