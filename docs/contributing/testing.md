# Testing

Consize uses automated tests to help catch regressions before changes are merged.

When making a change, run the tests that cover the part of the project you modified.

## Run the test suite

From the repository root:

```sh
pytest
```

If your environment uses a project-specific test command, follow the command documented by the repository.

## Run tests for a specific area

You can run a specific test file when working on one part of the codebase:

```sh
pytest path/to/test_file.py
```

You can also run an individual test:

```sh
pytest path/to/test_file.py::test_name
```

This is useful when iterating on a change and you do not want to run the entire test suite every time.

## Before opening a pull request

Before submitting a pull request:

1. Run the relevant tests.
2. Run the full test suite.
3. Check that documentation and configuration examples still work.
4. Review the changes with `git diff`.
5. Make sure unrelated files were not changed.

A typical workflow is:

```text
Make change
    ↓
Run focused tests
    ↓
Fix failures
    ↓
Run full test suite
    ↓
Review git diff
    ↓
Open pull request
```

## End-to-end tests

Some behavior cannot be fully tested with unit tests alone.

For tests that interact with a running Kubernetes environment, see [E2E Tests](e2e.md).

## When tests fail

Start by identifying whether the failure is related to your change.

Check:

* The failing test
* The error message
* Recent changes in the affected code
* Test configuration
* Kubernetes or external service dependencies

Avoid changing tests simply to make a failure disappear. Understand why the test is failing first.

## Next steps

* [E2E Tests](e2e.md)
* [Decisions](decisions.md)
* [How Consize Works](../concepts/architecture.md)
