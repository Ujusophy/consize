#!/usr/bin/env bash
set -euo pipefail
repo="$(git rev-parse --show-toplevel)"; oasdiff="${OASDIFF_BIN:-oasdiff}"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
printf 'not: [valid: OpenAPI\n' > "$tmp/invalid.yaml"
if (cd "$repo/ui" && npx --no-install redocly lint "$tmp/invalid.yaml" --config ../redocly.yaml) >/dev/null 2>&1; then echo "Invalid OpenAPI unexpectedly passed lint."; exit 1; fi
printf '// stale generated declaration\n' > "$tmp/stale-api.ts"
if GENERATED_API_FILE="$tmp/stale-api.ts" "$repo/scripts/api/check-generated.sh" >/dev/null 2>&1; then echo "Stale generated declarations unexpectedly passed."; exit 1; fi
if "$repo/scripts/api/compare-breaking.sh" invalid-revision-for-contract-test >/dev/null 2>&1; then echo "Invalid comparison revision unexpectedly passed."; exit 1; fi
if "$oasdiff" breaking --fail-on ERR "$repo/scripts/api/testdata/base.yaml" "$repo/scripts/api/testdata/breaking.yaml" >/dev/null 2>&1; then echo "Representative breaking change unexpectedly passed."; exit 1; fi
"$oasdiff" breaking --fail-on ERR "$repo/scripts/api/testdata/base.yaml" "$repo/scripts/api/testdata/compatible.yaml" >/dev/null
echo "Invalid schema, stale generation, invalid base, breaking change, and compatible change paths behave as expected."
