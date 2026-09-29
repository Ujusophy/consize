#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
runner="$root/scripts/security/run-gitleaks.sh"
config="$root/.gitleaks.toml"
temp="$(mktemp -d)"
trap 'rm -rf "$temp"' EXIT

git -C "$temp" init -q
git -C "$temp" config user.name "Consize Security Test"
git -C "$temp" config user.email "security-test@invalid.example"
cp "$config" "$temp/.gitleaks.toml"
mkdir -p "$temp/config"
printf 'TOKEN=replace-me\n' > "$temp/config/example.env"
git -C "$temp" add .
git -C "$temp" commit -qm "clean base"
base="$(git -C "$temp" rev-parse HEAD)"

synthetic_prefix="CONSIZE_TEST_"
synthetic_value="${synthetic_prefix}TOKEN_1234567890ABCDEFGHIJKLMN"
printf 'TOKEN=%s\n' "$synthetic_value" > "$temp/config/temporary.env"
git -C "$temp" add config/temporary.env
git -C "$temp" commit -qm "add disposable synthetic token"
secret_commit="$(git -C "$temp" rev-parse HEAD)"

(
  cd "$temp"
  SCAN_CONTEXT=pull_request GITLEAKS_BIN="${GITLEAKS_BIN:-gitleaks}" \
    "$runner" snapshot "$temp/active-snapshot.json" "" "$secret_commit"
)
grep -q '"path": "config/temporary.env"' "$temp/active-snapshot.json"
if grep -q 'CONSIZE_TEST_TOKEN_' "$temp/active-snapshot.json"; then
  echo "Snapshot report exposed the synthetic token." >&2
  exit 1
fi

git -C "$temp" rm -q config/temporary.env
git -C "$temp" commit -qm "remove disposable synthetic token"

(
  cd "$temp"
  SCAN_CONTEXT=pull_request GITLEAKS_BIN="${GITLEAKS_BIN:-gitleaks}" \
    "$runner" range "$temp/report.json" "$base"
)

grep -q 'consize-disposable-test-token' "$temp/report.json"
grep -q "${secret_commit:0:12}" "$temp/report.json"
if grep -q 'CONSIZE_TEST_TOKEN_' "$temp/report.json"; then
  echo "Normalized report exposed the synthetic token." >&2
  exit 1
fi

(
  cd "$temp"
  SCAN_CONTEXT=pull_request GITLEAKS_BIN="${GITLEAKS_BIN:-gitleaks}" \
    "$runner" snapshot "$temp/snapshot.json"
)
grep -q '"findings": \[\]' "$temp/snapshot.json"

if (cd "$temp" && GITLEAKS_BIN="${GITLEAKS_BIN:-gitleaks}" "$runner" range "$temp/invalid.json" deadbeef 2>/dev/null); then
  echo "Invalid comparison base unexpectedly passed." >&2
  exit 1
fi

# A shallow clone cannot establish the complete introduced commit range.
git clone -q --depth 1 "file://$temp" "$temp-shallow"
cp "$config" "$temp-shallow/.gitleaks.toml"
if (cd "$temp-shallow" && GITLEAKS_BIN="${GITLEAKS_BIN:-gitleaks}" "$runner" range "$temp/shallow.json" "$base" 2>/dev/null); then
  echo "Shallow history unexpectedly passed." >&2
  exit 1
fi
rm -rf "$temp-shallow"

echo "Gitleaks regression checks passed."
