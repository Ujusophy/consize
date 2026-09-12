# E2E Tests

End-to-end tests validate Consize against a real Kubernetes environment.

They are useful for testing behavior that cannot be fully covered by unit tests alone. Consize's E2E suite runs against a dedicated sandbox cluster, nightly and pre-release, using synthetic fixture workloads (never real customer data).

## When to use E2E tests

Use E2E tests when a change affects things such as:

* Kubernetes resources
* Controllers or collectors
* RBAC
* Runtime changes
* Helm deployment behavior
* Components communicating with each other

For changes that do not require a running Kubernetes environment, prefer the regular test suite, see [Testing](testing.md).

## Before running E2E tests

Make sure you have:

* A working Kubernetes cluster
* `kubectl` configured for the cluster
* The required Consize components deployed (see [Production Installation](../getting-started/installation.md))

Check your current Kubernetes context:

```sh
kubectl config current-context
```

Then verify that the cluster is reachable:

```sh
kubectl get nodes
```

## Running E2E tests

The engine is written in Go. E2E tests are built with an `e2e` build tag so they don't run as part of the regular `go test ./...` suite:

```sh
cd engine
go test -tags=e2e ./e2e/...
```

Use the test configuration provided by the repository rather than creating a separate local configuration unless the test requires it.

## What E2E tests verify

E2E tests use synthetic fixture workloads (a fixture exporter emitting CPU/memory at a known, fixed profile) so expected outcomes are deterministic.

**Happy path (compute):** deploy fixture workloads with inflated requests (e.g. 8Gi requested against 300Mi actually used), confirm Consize generates the expected rightsizing recommendation.

**Rollback path**, the scenario that proves the safety guarantees actually work: deploy a workload with a latency bug, apply a rightsizing change, the induced bug pushes error rate up 50%, the verifier returns `FAIL`, Consize automatically rolls back to the previous values, records the event as `rolled_back` with evidence, and fires an alert. The test asserts the rollout was actually restored, not just that a rollback was logged.

**Idempotency:** applying the same recommendation twice is a no-op the second time, since the workload is already at the target values.

## Keep E2E tests isolated

E2E tests should avoid modifying unrelated workloads or shared environments.

When possible:

* Use dedicated test namespaces
* Use predictable, synthetic test resources, never real customer data
* Clean up resources after the test
* Avoid production clusters
* Keep test credentials separate from production credentials

## Debugging failures

When an E2E test fails, check the Kubernetes environment first.

Useful commands include:

```sh
kubectl get pods -A
```

```sh
kubectl get events -A --sort-by=.lastTimestamp
```

For a specific namespace:

```sh
kubectl -n <namespace> get pods
```

Then inspect the logs of the affected component:

```sh
kubectl -n <namespace> logs <pod-name>
```

## Next steps

* [Testing](testing.md)
* [Decisions](decisions.md)
* [Production Installation](../getting-started/installation.md)