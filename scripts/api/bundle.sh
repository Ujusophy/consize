#!/usr/bin/env bash
set -euo pipefail
repo="$(git rev-parse --show-toplevel)"; output="${1:-$repo/openapi/dist}"; revision="${GITHUB_SHA:-$(git rev-parse HEAD)}"
mkdir -p "$output"
(cd "$repo/ui" && npx --no-install redocly bundle ../openapi/openapi.yaml --config ../redocly.yaml --output "$output/openapi.bundle.yaml")
if command -v sha256sum >/dev/null 2>&1; then checksum="$(sha256sum "$output/openapi.bundle.yaml" | awk '{print $1}')"; else checksum="$(shasum -a 256 "$output/openapi.bundle.yaml" | awk '{print $1}')"; fi
printf '{\n  "git_revision": "%s",\n  "sha256": "%s",\n  "source": "openapi/openapi.yaml"\n}\n' "$revision" "$checksum" > "$output/manifest.json"
echo "Bundled contract $checksum for Git revision $revision"
