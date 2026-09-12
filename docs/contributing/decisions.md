# Decisions

Consize makes technical decisions that affect its architecture, safety model, and developer experience.

The existing decisions, roughly 40 of them, covering everything from why sizing uses percentiles instead of averages to why v1 only downsizes, are recorded in the [decision log](https://github.com/consize-oss/consize/blob/main/docs/reference.md). Read it before proposing a change to the safety engine, sizing policy, or permissions model, it will often already explain why something works the way it does.

This page explains how contributors should think about *new* decisions and where to document them.

## Why document decisions?

Some changes are easy to reverse.

Others affect how Consize works for users and contributors for a long time.

Examples include:

* Changing the safety model
* Adding a new runtime component
* Changing Kubernetes permissions
* Introducing a new dependency
* Changing how recommendations are generated
* Changing how configuration works

Documenting these decisions gives future contributors the context behind the implementation.

## Before making a significant change

Start by understanding the existing design.

Read the relevant documentation and code before proposing a new approach.

For architecture-related changes, start with:

* [How Consize Works](../concepts/architecture.md)
* [The Safety Net](../concepts/safety-net.md)
* [Configuration](../reference/configuration.md)

## What to consider

When proposing a significant change, consider:

### Safety

Could this change allow Consize to make a change it should not make?

Does it affect Kubernetes permissions or the boundaries around automatic application?

### User experience

Does the change make Consize easier or harder to understand and operate?

Consider the experience of someone discovering Consize for the first time as well as someone operating it in production.

### Complexity

Does the change introduce unnecessary components, dependencies, or configuration?

Prefer the simplest design that solves the problem.

### Backward compatibility

Could the change break existing installations, configuration, or workflows?

If so, document the migration path.

### Observability

If something goes wrong, will operators have enough information to understand what happened?

## Document the decision

For significant architectural decisions, record:

1. **Context** — What problem are we solving?
2. **Decision** — What approach are we taking?
3. **Alternatives** — What other approaches were considered?
4. **Tradeoffs** — What are we gaining and giving up?
5. **Consequences** — What does this mean for users and contributors?

Keep the explanation focused on the decision rather than documenting every step of the discussion.

## Keep documentation current

When a decision changes how Consize works, update the relevant documentation at the same time.

A code change that changes user behavior should not leave the documentation behind.

## Next steps

* [Testing](testing.md)
* [E2E Tests](e2e.md)
* [How Consize Works](../concepts/architecture.md)
* [The Safety Net](../concepts/safety-net.md)