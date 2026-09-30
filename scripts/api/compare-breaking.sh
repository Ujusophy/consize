#!/usr/bin/env bash
set -euo pipefail
repo="$(git rev-parse --show-toplevel)"; base="${1:-}"; oasdiff="${OASDIFF_BIN:-oasdiff}"; bootstrap="$(cat "$repo/openapi/.bootstrap-base")"; tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
[[ -n "$base" ]] || { echo "A comparison base revision is required."; exit 2; }
git -C "$repo" cat-file -e "$base^{commit}" 2>/dev/null || { echo "Invalid or unavailable comparison base: $base"; exit 2; }
merge_base="$(git -C "$repo" merge-base "$base" HEAD)"; [[ -n "$merge_base" ]] || { echo "No merge base found for $base and HEAD."; exit 2; }
if git -C "$repo" cat-file -e "$merge_base:openapi/openapi.yaml" 2>/dev/null; then
  git -C "$repo" show "$merge_base:openapi/openapi.yaml" > "$tmp/base.yaml"
elif [[ "$merge_base" == "$bootstrap" ]]; then
  echo "Bootstrap comparison: $merge_base predates the canonical contract."; exit 0
else
  echo "Canonical contract is missing from comparison base $merge_base; refusing to pass."; exit 2
fi
"$oasdiff" breaking --fail-on ERR "$tmp/base.yaml" "$repo/openapi/openapi.yaml"
