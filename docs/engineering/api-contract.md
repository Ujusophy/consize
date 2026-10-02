# API contract workflow

## Purpose

`openapi/openapi.yaml` is the canonical contract for the HTTP API implemented by the v0.3 backend and consumed by the frontend. It prevents the handler responses, UI declarations, and integration documentation from becoming separate versions of the same API.

The contract describes behavior that exists today. Adding a future operation to the document does not create a handler and is not permitted as a substitute for implementation work.

## Namespaces

- **Operational API:** `/api/health` is unauthenticated and intended for process health checks and operators. It must not expose protected product data.
- **Product API:** the other current `/api/*` operations are authenticated product endpoints used by the UI and integrations. They are not being renamed to an invented `/api/v1/public` namespace during this ticket.

The project can introduce explicit public or versioned namespaces later through a reviewed compatibility decision. “Public” means supported for external consumers; it does not mean every internal or operational endpoint must share that namespace.

## Source and generated output

- Canonical source: `openapi/openapi.yaml`
- Lint policy: `redocly.yaml`
- Generated frontend declarations: `ui/lib/generated/api.ts`
- Compatibility scripts and fixtures: `scripts/api/`
- CI check name: `API contract`

The generated TypeScript file is committed so frontend builds do not require generation. It must never be edited manually.

## Local workflow

Install the locked frontend toolchain once:

```bash
npm --prefix ui ci
```

After changing the API contract:

```bash
npm --prefix ui run api:generate
(cd ui && npx --no-install redocly lint ../openapi/openapi.yaml --config ../redocly.yaml)
scripts/api/check-generated.sh
go test ./internal/api -run OpenAPI
```

Install the pinned compatibility checker and compare against the target branch:

```bash
GOBIN="$(pwd)/.tools" go install github.com/oasdiff/oasdiff@v1.29.1
OASDIFF_BIN="$(pwd)/.tools/oasdiff" scripts/api/test-contract.sh
OASDIFF_BIN="$(pwd)/.tools/oasdiff" scripts/api/compare-breaking.sh origin/main
```

Create a bundled artifact and revision manifest:

```bash
scripts/api/bundle.sh /tmp/consize-api-contract
```

The manifest records the Git revision and SHA-256 of the bundle. CI uploads both as evidence; generated distribution files are not committed.

## Change process

1. Change the backend behavior and canonical contract in the same branch.
2. Regenerate `ui/lib/generated/api.ts` with the locked command.
3. Update affected frontend consumers using the generated declarations.
4. Run lint, route tests, generation drift, compatibility tests, and the frontend build locally.
5. Request review for both backend/API behavior and affected frontend usage.
6. Merge only after the required `API contract` check succeeds.

Repository owners must configure branch protection to require the emitted `API contract` check. CODEOWNERS marks the contract surfaces, but repository settings determine the number and independence of required reviewers. Until separate backend and frontend teams exist in CODEOWNERS, the pull request must explicitly record both review perspectives.

## Compatibility policy

CI fetches complete history and calculates a merge base with the pull request target. A missing or invalid Git base fails. The first contract PR has one narrow bootstrap allowance tied to the exact pre-contract commit in `openapi/.bootstrap-base`; after this change reaches `main`, comparisons use the canonical source from the merge base.

`oasdiff v1.29.1` blocks definite `ERR`-level breaking changes. `WARN` findings remain non-blocking because a machine cannot determine every consumer impact, but CI emits each warning as a GitHub annotation and includes the full compatibility report in the job summary. An API-contract reviewer must acknowledge any warning in the pull-request review before merge. A breaking change needs an explicit versioning and migration decision; it must not be hidden by changing the comparison base.

## What automation proves

The gate proves that the description is structurally valid, required operation metadata and response headers exist, generated declarations are deterministic and current, documented routes match the implemented route inventory, and detected incompatible schema changes block CI.

It does **not** prove runtime semantics, authorization correctness, latency, idempotency, ordering, side effects, error meaning, or that every response produced by every code path conforms to the schema. Backend tests, integration tests, security review, and human contract review remain required. Future work may add runtime request/response validation, but it must not silently change production behavior.

## Troubleshooting

- **Stale generated declarations:** run `npm --prefix ui run api:generate` and commit the result.
- **Invalid comparison base:** fetch full history and pass a reachable commit or branch. Do not bypass the check.
- **Lint error:** fix the source contract; do not create a broad ignore file.
- **Bundle checksum changed:** expected when the contract changes. Confirm the source diff and generated declarations are part of the same review.
- **Compatibility warning:** read the `API compatibility review` section in the job summary, explain the expected consumer impact in the pull request, and obtain acknowledgement from an API-contract reviewer before merge. Do not treat a green check as automatic approval of WARN findings.
