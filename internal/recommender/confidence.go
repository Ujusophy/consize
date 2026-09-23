package recommender

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/consize-oss/consize/internal/bootstrap"
	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
)

// ConfidenceReport is an evidence-quality assessment, not a probability of safety.
type ConfidenceReport struct {
	Model           string    `json:"model"`
	Label           string    `json:"label"`
	Profile         string    `json:"profile"`
	Signal          string    `json:"signal"`
	RequiredHistory string    `json:"required_history"`
	ObservedHistory string    `json:"observed_history"`
	CoverageRatio   float64   `json:"coverage_ratio"`
	PeakToP95Ratio  float64   `json:"peak_to_p95_ratio"`
	CollectedAt     time.Time `json:"collected_at"`
	Reasons         []string  `json:"reasons"`
}

func AssessConfidence(cfg bootstrap.RecommenderConfig, res resource.Resource, snapshot plugin.MetricsSnapshot, now time.Time) ConfidenceReport {
	cfg = withDefaults(cfg)
	signal := strings.TrimSuffix(cfg.MetricKey, "_p95")
	report := ConfidenceReport{Model: "history-coverage-v1", Label: "low", Profile: cfg.ConfidenceProfile, Signal: signal, CollectedAt: snapshot.CollectedAt, Reasons: []string{}}
	minHistory, highHistory := 24*time.Hour, 7*24*time.Hour
	if res.Environment == resource.EnvProduction || res.Criticality == resource.CriticalityHigh {
		minHistory = 7 * 24 * time.Hour
		highHistory = 14 * 24 * time.Hour
	}
	if cfg.ConfidenceProfile == "lab" {
		minHistory = 30 * time.Minute
		if res.Environment != resource.EnvDevelopment || res.Criticality == resource.CriticalityHigh {
			report.Reasons = append(report.Reasons, "lab profile is restricted to non-critical development resources")
		}
	}
	if cfg.ConfidenceProfile != "standard" && cfg.ConfidenceProfile != "lab" {
		report.Reasons = append(report.Reasons, "unknown confidence profile")
	}
	report.RequiredHistory = minHistory.String()
	if snapshot.ResourceID != res.ID || snapshot.PluginID != cfg.MetricsPluginID {
		report.Reasons = append(report.Reasons, "metrics identity does not match resource or configured plugin")
	}
	if snapshot.CollectedAt.IsZero() || now.Sub(snapshot.CollectedAt) > 2*time.Minute || snapshot.CollectedAt.After(now.Add(time.Minute)) {
		report.Reasons = append(report.Reasons, "metrics collection timestamp is stale or invalid")
	}
	window, err := time.ParseDuration(snapshot.Window)
	if err != nil || window <= 0 {
		report.Reasons = append(report.Reasons, "metrics window is invalid")
	}
	coverage, ok := snapshot.Coverage[signal]
	if !ok || coverage.Step <= 0 || coverage.FirstSample.IsZero() || coverage.LastSample.IsZero() || coverage.LastSample.Before(coverage.FirstSample) {
		report.Reasons = append(report.Reasons, "timestamped metric coverage is missing or invalid")
		return report
	}
	span := coverage.LastSample.Sub(coverage.FirstSample)
	report.ObservedHistory = span.String()
	if window < minHistory || span < minHistory-coverage.Step || coverage.Step > minHistory/24 {
		report.Reasons = append(report.Reasons, fmt.Sprintf("insufficient actual history: need %s with at least 24 sampling intervals", minHistory))
	}
	expected := int(window/coverage.Step) + 1
	if coverage.ExpectedPoints != expected || coverage.ExpectedPoints < 2 || coverage.ActualPoints < 2 || coverage.ActualPoints > coverage.ExpectedPoints {
		report.Reasons = append(report.Reasons, "inconsistent sample counts")
	} else {
		report.CoverageRatio = float64(coverage.ActualPoints) / float64(coverage.ExpectedPoints)
	}
	if report.CoverageRatio < 0.90 {
		report.Reasons = append(report.Reasons, "coverage below required 90%")
	}
	if coverage.LastSample.After(snapshot.CollectedAt) || snapshot.CollectedAt.Sub(coverage.LastSample) > 2*coverage.Step || coverage.FirstSample.Before(snapshot.CollectedAt.Add(-window)) || coverage.FirstSample.After(snapshot.CollectedAt.Add(-window+2*coverage.Step)) {
		report.Reasons = append(report.Reasons, "coverage timestamps do not span the claimed window")
	}
	if coverage.MaxGap <= 0 || coverage.MaxGap > 3*coverage.Step {
		report.Reasons = append(report.Reasons, "unknown or excessive gaps between samples")
	}
	p95, p95OK := asFloat(snapshot.Signals[cfg.MetricKey])
	peak, peakOK := asFloat(snapshot.Signals[signal+"_max"])
	if !p95OK || !peakOK || math.IsNaN(p95) || math.IsInf(p95, 0) || math.IsNaN(peak) || math.IsInf(peak, 0) || p95 <= 0 || peak < p95 {
		report.Reasons = append(report.Reasons, "valid p95 and peak measurements are required")
	} else {
		report.PeakToP95Ratio = peak / p95
		if report.PeakToP95Ratio > 2 {
			report.Reasons = append(report.Reasons, "peak exceeds twice p95; burst-sensitive workload requires review")
		}
	}
	if len(report.Reasons) > 0 {
		return report
	}
	report.Label = "medium"
	report.Reasons = append(report.Reasons, "history, freshness, coverage and peak checks passed")
	if cfg.ConfidenceProfile == "lab" {
		report.Reasons = append(report.Reasons, "lab assessment only; not representative production history")
		return report
	}
	if window >= highHistory && span >= highHistory-coverage.Step && report.CoverageRatio >= 0.98 {
		report.Label = "high"
	} else {
		report.Reasons = append(report.Reasons, fmt.Sprintf("high confidence requires %s and at least 98%% coverage", highHistory))
	}
	report.Reasons = append(report.Reasons, "known demand cycles, deployment history and incidents are not yet evaluated")
	return report
}
