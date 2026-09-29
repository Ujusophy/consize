#!/usr/bin/env bash
set -euo pipefail

version="8.30.1"
destination="${1:-${RUNNER_TEMP:-/tmp}/gitleaks}"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
machine="$(uname -m)"

case "${os}/${machine}" in
  linux/x86_64) asset="linux_x64"; checksum="551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb" ;;
  linux/aarch64|linux/arm64) asset="linux_arm64"; checksum="e4a487ee7ccd7d3a7f7ec08657610aa3606637dab924210b3aee62570fb4b080" ;;
  darwin/arm64) asset="darwin_arm64"; checksum="b40ab0ae55c505963e365f271a8d3846efbc170aa17f2607f13df610a9aeb6a5" ;;
  *) printf 'Unsupported Gitleaks platform: %s/%s\n' "$os" "$machine" >&2; exit 2 ;;
esac

archive="$(mktemp)"
trap 'rm -f "$archive"' EXIT
url="https://github.com/gitleaks/gitleaks/releases/download/v${version}/gitleaks_${version}_${asset}.tar.gz"
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 "$url" -o "$archive"

actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
if [[ "$actual" != "$checksum" ]]; then
  printf 'Gitleaks archive checksum mismatch for %s\n' "$asset" >&2
  exit 3
fi

install_dir="$(dirname "$destination")"
mkdir -p "$install_dir"
extracted="$(mktemp -d)"
trap 'rm -f "$archive"; rm -rf "$extracted"' EXIT
tar -xzf "$archive" -C "$extracted" gitleaks
mv "$extracted/gitleaks" "$destination"
chmod 0755 "$destination"
if [[ "$("$destination" version)" != "$version" ]]; then
  printf 'Installed Gitleaks version did not match %s\n' "$version" >&2
  exit 4
fi
printf 'Installed Gitleaks %s with verified SHA-256 checksum.\n' "$version"
