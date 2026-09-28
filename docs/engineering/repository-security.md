# Repository Security

The `Repository security` CI check protects the material that enters the Consize repository and the trust boundary around pull requests. It complements secret and dependency scanners. It does not claim that pattern matching can prove that a repository contains no secret or unsafe configuration.

## What the gate checks

The scanner reads blobs from the Git index at the checked-out revision. It does not scan ignored developer files or print matched values. It checks:

- tracked private environment files, private keys, kubeconfigs, common cloud credentials and credential archives;
- non-placeholder credential assignments in source, Dockerfiles and Compose files;
- sensitive `NEXT_PUBLIC_*` names, server-only environment access in client components, and client imports from server-only modules;
- required `.gitignore` protections and, when container definitions exist, required `.dockerignore` protections;
- local state, audit JSONL, build output, coverage, tool caches and generated artifacts that should not be tracked;
- unexpected executable files, executable modes, symlinks, submodules and oversized files;
- explicit workflow permissions, immutable action references, privileged pull-request events and direct shell interpolation of untrusted event fields.

The scanner creates a redacted normalized report. The common security-policy evaluator decides whether findings block, require triage, or match a separately approved security exception. A scanner error or malformed report is an incomplete security check, not a clean result.

## Run it locally

```bash
go test ./internal/repositorysecurity ./cmd/repository-security

go run ./cmd/repository-security \
  --context build \
  --output /tmp/repository-security-report.json

go run ./cmd/security-policy evaluate \
  --policy .github/security/policy.json \
  --exceptions .github/security/exceptions.json \
  --report /tmp/repository-security-report.json
```

The scan evaluates committed Git-index content. Commit a fixture on a disposable branch when proving a history-sensitive failure. Never use a working credential.

## Reviewed declarations

`.github/security/repository-allowlist.json` is for narrow non-secret examples, scanner fixtures, approved generated assets and approved tools. A declaration must identify one rule, one exact path and one exact SHA-256 digest. It also requires an owner, independent approver, review time, expiry and meaningful reason.

Declarations do not disable scanning. A file change invalidates its declaration, expired declarations fail validation, and declarations that no longer match a finding fail as stale. Broad paths, globs and directory exclusions are rejected. Live credentials cannot be declared safe.

Use the separate `.github/security/exceptions.json` process for temporary risk decisions produced by the common evaluator. Repository declarations are only for content that is intentionally non-secret or a specifically reviewed artifact.

## Human review remains required

Reviewers must still examine configuration intent, data flow and trust boundaries. In particular, verify that:

- examples contain no real tenant names, endpoints, tokens or encoded credentials;
- server-only values cannot reach browser bundles through indirect imports or generated configuration;
- Docker build contexts contain only required files and production credentials are injected at runtime;
- workflow changes do not expose secrets to forks, execute untrusted code with write permissions, or broaden token permissions;
- approved binaries and large assets have a documented source, checksum, owner and removal or review date.

Patterns can miss obfuscated, encrypted, split, dynamically assembled or unfamiliar credential formats. This gate is one control in the repository security baseline, not proof of absence.

## Failure ownership

The author removes ordinary violations. Backend/integration engineering owns server configuration and runtime packaging findings. Frontend engineering owns browser-exposure findings. Repository owners review workflow and artifact trust changes. The security approver reviews declarations and exceptions. High and critical findings block merge until fixed or handled under the common policy.

Redacted reports are retained by CI for 14 days. Reports may identify paths and rule IDs but must not contain matched values. Suspected live credentials follow `SECURITY.md`: revoke or rotate first, preserve redacted evidence, then remove the repository copy.
