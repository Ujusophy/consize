# E2E Tests

End-to-end tests validate Consize against a real Kubernetes environment.

They are useful for testing behavior that cannot be fully covered by unit tests alone.

## When to use E2E tests

Use E2E tests when a change affects things such as:

* Kubernetes resources
* Controllers or collectors
* RBAC
* Runtime changes
* Helm deployment behavior
* Components communicating with each other

For changes that do not require a running Kubernetes environment, prefer the regular test suite.

## Before running E2E tests

Make sure you have:

* A working Kubernetes cluster
* `kubectl` configured for the cluster
* The required Consize components available
* The dependencies required by the test environment

Check your current Kubernetes context:

```sh id="x8q4mt"
kubectl config current-context
```

Then verify that the cluster is reachable:

```sh id="k2v7pc"
kubectl get nodes
```

## Running E2E tests

Run the repository's E2E test command from the project root.

If the tests are implemented with pytest, a specific E2E test can be run with:

```sh id="m5n9rx"
pytest path/to/e2e_test.py
```

Use the test configuration provided by the repository rather than creating a separate local configuration unless the test requires it.

## What E2E tests should verify

An E2E test should verify behavior across the system rather than testing an individual function.

For example:

```text id="j6p3wv"
Deploy Consize
      ↓
Create test workload
      ↓
Collect workload data
      ↓
Generate recommendation
      ↓
Apply change
      ↓
Verify resulting state
```

## Keep E2E tests isolated

E2E tests should avoid modifying unrelated workloads or shared environments.

When possible:

* Use dedicated test namespaces
* Use predictable test resources
* Clean up resources after the test
* Avoid production clusters
* Keep test credentials separate from production credentials

## Debugging failures

When an E2E test fails, check the Kubernetes environment first.

Useful commands include:

```sh id="r4x8nd"
kubectl get pods -A
```

```sh id="u7m2kp"
kubectl get events -A --sort-by=.lastTimestamp
```

For a specific namespace:

```sh id="c9q5vt"
kubectl -n <namespace> get pods
```

Then inspect the logs of the affected component:

```sh id="a3k6yw"
kubectl -n <namespace> logs <pod-name>
```

## Next steps

* [Testing](testing.md)
* [Decisions](decisions.md)
* [Production Installation](../getting-started/installation.md)
