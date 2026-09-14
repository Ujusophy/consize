# API Reference

Consize exposes a REST API (Go, under `/api/v1`) that backs the UI. It's documented here mainly so you can script against it or understand what the UI is calling, not as a standalone product surface, it isn't versioned or supported as a separate integration point yet.

## Base URL and health

The API is reachable at `http://<consize-api-service>:8080` inside the cluster (see [Production Installation](../getting-started/installation.md#7-verify-the-installation) for port-forwarding it locally).

| Endpoint | Auth | Description |
|---|---|---|
| `GET /healthz` | None | Process liveness |
| `GET /readyz` | None | Ready to serve traffic, `{"status":"ready"}` when healthy |

## Authentication

Auth is optional and off by default (`CONSIZE_AUTH_REQUIRED=false`), see [Configuration](configuration.md#authentication). When enabled, there are three roles: **viewer**, **operator**, **admin**, each strictly more permissive than the last.

| Endpoint | Description |
|---|---|
| `POST /api/v1/auth/login` | Log in, sets a session cookie |
| `POST /api/v1/auth/logout` | Revoke the current session |
| `POST /api/v1/auth/setup` | First-run wizard: creates the first admin account, only works while no users exist |
| `GET /api/v1/auth/me` | Current session info. Returns `auth_enabled: false` when auth is off, so the UI degrades gracefully instead of asking for a login it doesn't need. |

## Reads — any authenticated user (viewer and above)

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/teams` | List teams |
| GET | `/api/v1/workloads` | List workloads |
| GET | `/api/v1/workloads/{id}` | Get a workload |
| GET | `/api/v1/workloads/{id}/series` | Workload usage time series |
| GET | `/api/v1/recommendations` | List recommendations (`workload_id`, `status`, `limit`, `offset` query params) |
| GET | `/api/v1/savings` | Realized and projected savings |
| GET | `/api/v1/system/status` | Collector/analysis/verifier health |
| GET | `/api/v1/alerting/config` | Current alerting configuration |
| GET | `/api/v1/integrations/github` | Current GitHub integration config |
| GET | `/api/v1/reports/config` | Current savings-report configuration |
| GET | `/api/v1/reports/savings` | Savings report data |
| GET | `/api/v1/applies` | Apply event history (`workload_id`, `apply_event_id`, `result` query params) |
| GET | `/api/v1/verification-runs` | Verification run history |
| GET | `/api/v1/cost-opportunities` | List detected cloud waste |

## Writes — operator or admin

Applying anything requires at least the **operator** role. The server checks this, the client can't self-report a role.

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/recommendations/{id}/apply` | Apply a rightsizing recommendation directly |
| POST | `/api/v1/recommendations/{id}/iac-pr` | Open an IaC pull request for a recommendation instead |
| POST | `/api/v1/cost-opportunities/scan` | Trigger a cloud waste scan |
| POST | `/api/v1/cost-opportunities/{id}/apply` | Clean up a detected waste item directly |
| POST | `/api/v1/cost-opportunities/{id}/iac-pr` | Open an IaC pull request for a waste cleanup instead |

## Governance — admin only

Ownership and integration configuration, not routine operation.

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/teams` | Create a team |
| PATCH | `/api/v1/teams/{id}` | Update a team |
| PUT | `/api/v1/workloads/{id}/team` | Assign a workload to a team |
| DELETE | `/api/v1/workloads/{id}/team` | Unassign a workload's team |
| PUT | `/api/v1/alerting/config` | Update alerting configuration |
| POST | `/api/v1/alerting/test` | Send a test alert |
| PUT | `/api/v1/integrations/github` | Update GitHub integration config |
| PUT | `/api/v1/reports/config` | Update savings-report configuration |
| POST | `/api/v1/reports/send` | Send a savings report immediately |

## Next steps

* [Configuration](configuration.md)
* [The Safety Net](../concepts/safety-net.md)
* [How Consize Works](../concepts/architecture.md)