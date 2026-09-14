#!/bin/sh
# Keeps the Helm --version string in README.md and _snippets/helm-install-oci.sh
# in sync with charts/consize/Chart.yaml, the one place a release actually
# bumps the version.
#
# Usage:
#   scripts/sync-version.sh          fixes any mismatches in place
#   scripts/sync-version.sh --check  exits 1 and prints a diff if anything's
#                                     out of sync, without changing files
#                                     (this is what CI runs on every push/PR)
set -eu

cd "$(dirname "$0")/.."

CHART_FILE="charts/consize/Chart.yaml"
CHART_VERSION=$(grep -m1 '^version:' "$CHART_FILE" | awk '{print $2}')

if [ -z "$CHART_VERSION" ]; then
  echo "Could not read version from $CHART_FILE" >&2
  exit 1
fi

TARGETS="README.md _snippets/helm-install-oci.sh"
MODE="${1:-fix}"
EXIT_CODE=0

for FILE in $TARGETS; do
  if [ ! -f "$FILE" ]; then
    echo "Skipping $FILE (not found)" >&2
    continue
  fi

  CURRENT=$(grep -m1 -- '--version ' "$FILE" | sed -E 's/.*--version ([0-9]+\.[0-9]+\.[0-9]+).*/\1/')

  if [ "$CURRENT" = "$CHART_VERSION" ]; then
    continue
  fi

  if [ "$MODE" = "--check" ]; then
    echo "OUT OF SYNC: $FILE has --version $CURRENT, Chart.yaml has $CHART_VERSION" >&2
    EXIT_CODE=1
  else
    sed -i.bak -E "s/(--version )[0-9]+\.[0-9]+\.[0-9]+/\1$CHART_VERSION/" "$FILE"
    rm -f "$FILE.bak"
    echo "Updated $FILE to --version $CHART_VERSION"
  fi
done

if [ "$MODE" = "--check" ] && [ "$EXIT_CODE" -eq 0 ]; then
  echo "README.md and _snippets/helm-install-oci.sh match Chart.yaml ($CHART_VERSION)"
fi

exit $EXIT_CODE