#!/usr/bin/env bash
set -euo pipefail

mode="${1:?usage: run-gitleaks.sh snapshot|range|history OUTPUT [BASE] [HEAD]}"
output="${2:?normalized output path is required}"
base="${3:-}"
head="${4:-HEAD}"
gitleaks="${GITLEAKS_BIN:-gitleaks}"
repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
raw="$(mktemp)"
snapshot=""
trap 'rm -f "$raw"; [[ -z "$snapshot" ]] || rm -rf "$snapshot"' EXIT

git rev-parse --is-inside-work-tree >/dev/null
git cat-file -e "${head}^{commit}" 2>/dev/null || { echo "Invalid scan head commit." >&2; exit 20; }
context="${SCAN_CONTEXT:-build}"
version="$(git rev-parse "$head")"
strip_prefix=""

set +e
case "$mode" in
  snapshot)
    snapshot="$(mktemp -d)"
    strip_prefix="$snapshot/"
    git archive "$head" | tar -xf - -C "$snapshot" || exit 21
    "$gitleaks" dir --no-banner --redact=100 --config .gitleaks.toml \
      --report-format json --report-path "$raw" "$snapshot"
    scanner_status=$?
    ;;
  range)
    [[ -n "$base" ]] || { echo "Range mode requires a base commit." >&2; exit 22; }
    [[ "$(git rev-parse --is-shallow-repository)" == "false" ]] || { echo "Git history is shallow; refusing an incomplete range scan." >&2; exit 23; }
    git cat-file -e "${base}^{commit}" 2>/dev/null || { echo "Base commit is unavailable." >&2; exit 24; }
    merge_base="$(git merge-base "$base" "$head")" || { echo "No valid merge base." >&2; exit 25; }
    [[ -n "$merge_base" ]] || { echo "No valid merge base." >&2; exit 25; }
    [[ "$(git rev-list --count "${merge_base}..${head}")" -gt 0 ]] || { echo "Comparison range contains no introduced commits." >&2; exit 26; }
    "$gitleaks" git --no-banner --redact=100 --config .gitleaks.toml \
      --report-format json --report-path "$raw" --log-opts="${merge_base}..${head}" .
    scanner_status=$?
    ;;
  history)
    [[ "$(git rev-parse --is-shallow-repository)" == "false" ]] || { echo "Git history is shallow; refusing an incomplete history scan." >&2; exit 27; }
    "$gitleaks" git --no-banner --redact=100 --config .gitleaks.toml \
      --report-format json --report-path "$raw" .
    scanner_status=$?
    ;;
  *) echo "Unsupported secret scan mode: $mode" >&2; exit 2 ;;
esac
set -e

# Gitleaks uses 1 for findings. Any other non-zero status is a scanner failure.
if [[ "$scanner_status" -ne 0 && "$scanner_status" -ne 1 ]]; then
  echo "Gitleaks did not complete successfully." >&2
  exit 30
fi
[[ -f "$raw" ]] || printf '[]\n' > "$raw"

go -C "$repository_root" run ./cmd/gitleaks-report \
  --input "$raw" \
  --output "$output" \
  --context "$context" \
  --version "$version" \
  --strip-prefix "$strip_prefix" \
  --scan-id "gitleaks-${mode}-${version}"
