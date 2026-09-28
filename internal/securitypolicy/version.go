package securitypolicy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semanticVersionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)

type semanticVersion struct {
	major      int
	minor      int
	patch      int
	prerelease []string
}

func validateVersionRange(constraint string) error {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return fmt.Errorf("affected_version_range is required")
	}
	if strings.ContainsAny(constraint, "*?[]{}") || strings.EqualFold(constraint, "all") {
		return fmt.Errorf("affected_version_range must not contain a wildcard or global scope")
	}
	if strings.HasPrefix(constraint, "exact:") {
		value := strings.TrimPrefix(constraint, "exact:")
		if value == "" || strings.ContainsAny(value, " \t\r\n") {
			return fmt.Errorf("exact version constraint must contain one non-whitespace value")
		}
		return nil
	}
	for _, term := range strings.Fields(constraint) {
		_, versionText := splitComparator(term)
		if _, err := parseSemanticVersion(versionText); err != nil {
			return fmt.Errorf("invalid version constraint %q: %w", term, err)
		}
	}
	return nil
}

func versionMatches(constraint, candidate string) bool {
	if strings.HasPrefix(constraint, "exact:") {
		return strings.TrimPrefix(constraint, "exact:") == candidate
	}
	want, err := parseSemanticVersion(candidate)
	if err != nil {
		return false
	}
	for _, term := range strings.Fields(constraint) {
		operator, versionText := splitComparator(term)
		limit, err := parseSemanticVersion(versionText)
		if err != nil || !compareVersion(want, limit, operator) {
			return false
		}
	}
	return true
}

func splitComparator(term string) (string, string) {
	for _, operator := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(term, operator) {
			return operator, strings.TrimPrefix(term, operator)
		}
	}
	return "=", term
}

func parseSemanticVersion(value string) (semanticVersion, error) {
	matches := semanticVersionPattern.FindStringSubmatch(value)
	if matches == nil {
		return semanticVersion{}, fmt.Errorf("%q is not a supported semantic version", value)
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return semanticVersion{}, fmt.Errorf("major version is outside the supported numeric range")
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return semanticVersion{}, fmt.Errorf("minor version is outside the supported numeric range")
	}
	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return semanticVersion{}, fmt.Errorf("patch version is outside the supported numeric range")
	}
	var prerelease []string
	if matches[4] != "" {
		prerelease = strings.Split(matches[4], ".")
		for _, identifier := range prerelease {
			if identifier == "" {
				return semanticVersion{}, fmt.Errorf("prerelease contains an empty identifier")
			}
			if len(identifier) > 1 && identifier[0] == '0' && numeric(identifier) {
				return semanticVersion{}, fmt.Errorf("numeric prerelease identifiers must not contain leading zeroes")
			}
		}
	}
	return semanticVersion{major: major, minor: minor, patch: patch, prerelease: prerelease}, nil
}

func compareVersion(left, right semanticVersion, operator string) bool {
	comparison := left.compare(right)
	switch operator {
	case "=":
		return comparison == 0
	case ">":
		return comparison > 0
	case ">=":
		return comparison >= 0
	case "<":
		return comparison < 0
	case "<=":
		return comparison <= 0
	default:
		return false
	}
}

func (left semanticVersion) compare(right semanticVersion) int {
	for _, pair := range [][2]int{{left.major, right.major}, {left.minor, right.minor}, {left.patch, right.patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	if len(left.prerelease) == 0 && len(right.prerelease) == 0 {
		return 0
	}
	if len(left.prerelease) == 0 {
		return 1
	}
	if len(right.prerelease) == 0 {
		return -1
	}
	for index := 0; index < len(left.prerelease) && index < len(right.prerelease); index++ {
		leftPart, rightPart := left.prerelease[index], right.prerelease[index]
		if leftPart == rightPart {
			continue
		}
		leftNumeric, rightNumeric := numeric(leftPart), numeric(rightPart)
		if leftNumeric && rightNumeric {
			if len(leftPart) < len(rightPart) || len(leftPart) == len(rightPart) && leftPart < rightPart {
				return -1
			}
			return 1
		}
		if leftNumeric {
			return -1
		}
		if rightNumeric {
			return 1
		}
		if leftPart < rightPart {
			return -1
		}
		return 1
	}
	if len(left.prerelease) < len(right.prerelease) {
		return -1
	}
	if len(left.prerelease) > len(right.prerelease) {
		return 1
	}
	return 0
}

func numeric(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
