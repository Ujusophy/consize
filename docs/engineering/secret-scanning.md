# Secret Scanning

Consize treats a detected credential as a blocking security event. The Secret
scan CI gate uses Gitleaks 8.30.1 to inspect both the current tracked source and
every commit introduced by a pull request. This second scan matters because a
credential remains exposed in Git history even when a later commit deletes it.

## What the gate scans

For pull requests and merge queues, CI checks out complete history and scans:

1. an archive of the current `HEAD`, which contains tracked files only; and
2. the complete `merge-base..HEAD` commit range.

For pushes to `main`, the gate scans the current tracked source and the commits
between the previous and current revisions. A missing commit, shallow checkout,
all-zero push base, empty comparison, or missing merge base fails the job rather
than producing a misleading clean result.

Repository owners must also run a full reachable-history scan when this control
is introduced and after any approved history rewrite:

```bash
scripts/security/install-gitleaks.sh /tmp/gitleaks
GITLEAKS_BIN=/tmp/gitleaks SCAN_CONTEXT=scheduled_scan \
  scripts/security/run-gitleaks.sh history /tmp/gitleaks-history.json
go run ./cmd/security-policy evaluate \
  --policy .github/security/policy.json \
  --exceptions .github/security/exceptions.json \
  --report /tmp/gitleaks-history.json \
  --output /tmp/gitleaks-history-decision.json
```

## Tool integrity and output handling

The installer downloads a pinned official Gitleaks release over HTTPS and
verifies a platform-specific SHA-256 digest before extraction. Version and
checksums are reviewed in `scripts/security/install-gitleaks.sh`; updates must
change all three together and cite the corresponding official release.

Gitleaks runs with full redaction. Its temporary raw report is deleted at the
end of the scan and is never uploaded. `cmd/gitleaks-report` rejects unknown or
unredacted scanner fields and creates the common finding format using only:

- rule ID;
- repository-relative path and line;
- abbreviated commit ID;
- a stable fingerprint derived from metadata.

CI uploads only normalized redacted reports and policy decisions. Scanner
errors, invalid JSON, absent reports, and normalization errors fail the gate.

## Placeholders and false positives

`.gitleaks.toml` permits a short list of unmistakable placeholder values such
as `replace-me` and `not-a-secret`. It does not exclude environment files,
source directories, generated reports, or whole rules.

A real false positive must be scanned first and then recorded as an exact
`false_positive` entry in `.github/security/exceptions.json`. The entry needs a
rule ID, redacted fingerprint, exact component and path, owner, independent
approver when required, evidence that the value cannot authenticate, expiry,
review date, and remediation issue. The common policy validator rejects broad,
expired, malformed, and automatically renewed exceptions. Never place the
matched value in an exception or issue.

## Live-secret response

If a finding may be a working credential:

1. revoke or rotate it immediately;
2. review provider activity, scope, and downstream copies;
3. remove it from current source and CI or local configuration;
4. determine whether reachable Git history contains it;
5. obtain repository-owner approval before rewriting published history;
6. record a redacted incident outcome and rotate related credentials when
   compromise cannot be excluded.

Deleting a file or commit does not revoke a credential. A live-secret exposure
cannot be accepted through the ordinary exception registry.

## Local verification

```bash
scripts/security/install-gitleaks.sh /tmp/gitleaks
GITLEAKS_BIN=/tmp/gitleaks go test ./internal/gitleaksreport ./cmd/gitleaks-report
GITLEAKS_BIN=/tmp/gitleaks scripts/security/test-gitleaks.sh
GITLEAKS_BIN=/tmp/gitleaks SCAN_CONTEXT=build \
  scripts/security/run-gitleaks.sh snapshot /tmp/gitleaks-snapshot.json
```

The regression script creates a disposable repository, commits a synthetic
non-credential token, deletes it in a later commit, and proves that the range
scan still finds it without exposing the value. It also proves that documented
placeholders pass and invalid or shallow comparison ranges fail.
