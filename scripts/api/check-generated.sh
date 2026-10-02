#!/usr/bin/env bash
set -euo pipefail
repo="$(git rev-parse --show-toplevel)"
generated="${GENERATED_API_FILE:-$repo/ui/lib/generated/api.ts}"
first="$(mktemp)"; second="$(mktemp)"
trap 'rm -f "$first" "$second"' EXIT
(cd "$repo/ui" && npx --no-install openapi-typescript ../openapi/openapi.yaml -o "$first" && npx --no-install openapi-typescript ../openapi/openapi.yaml -o "$second")
cmp -s "$first" "$second" || { echo "OpenAPI type generation is not deterministic."; exit 1; }
cmp -s "$first" "$generated" || { echo "Generated API declarations are stale. Run: npm --prefix ui run api:generate"; diff -u "$generated" "$first" || true; exit 1; }
echo "Generated API declarations are current and deterministic."
