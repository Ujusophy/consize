package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/consize-oss/consize/internal/gitleaksreport"
)

func main() {
	input := flag.String("input", "", "redacted Gitleaks JSON report")
	output := flag.String("output", "", "normalized report destination")
	context := flag.String("context", "build", "security scan context")
	version := flag.String("version", "", "scanned Git revision")
	scanID := flag.String("scan-id", "", "unique scan identity")
	stripPrefix := flag.String("strip-prefix", "", "path prefix removed from scanner findings")
	flag.Parse()
	if *input == "" || *output == "" || *version == "" || *scanID == "" {
		fail("input, output, version, and scan-id are required")
	}
	source, err := os.Open(*input)
	if err != nil {
		fail(err.Error())
	}
	defer source.Close()
	report, err := gitleaksreport.Normalize(source, gitleaksreport.Options{Context: *context, Version: *version, ScanID: *scanID, Now: time.Now(), StripPrefix: *stripPrefix})
	if err != nil {
		fail(err.Error())
	}
	destination, err := os.Create(*output)
	if err != nil {
		fail(err.Error())
	}
	encoder := json.NewEncoder(destination)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		destination.Close()
		fail(err.Error())
	}
	if err := destination.Close(); err != nil {
		fail(err.Error())
	}
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
