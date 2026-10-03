# ADR 0001: Universal Resource Identity

- Status: Accepted
- Date: 2026-10-02

## Context

Consize must govern different resource types through one lifecycle. The earlier model accepted arbitrary IDs and allowed incomplete records, making cross-account collisions, silent identity mutation, and duplicated provider models possible.

## Decision

The core resource model is provider-neutral. New IDs are deterministically derived from provider, account or cluster, location, resource type, and provider resource ID. Mutable operational data never participates in identity.

The registry validates identity, derives support status, enforces lifecycle transitions, rejects stale observations, and persists one model version. Provider-specific attributes remain extension data. Kubernetes discovery supplies its cluster and location through configuration rather than adding Kubernetes-only mandatory fields to the core.

Legacy noncanonical IDs are preserved when canonical rediscovery identifies the same object. This temporary alpha compatibility rule protects existing references without permitting general duplicate IDs.

## Consequences

- Every discovery integration must supply an explicit provider boundary and location.
- Incomplete legacy records fail startup instead of receiving guessed identity.
- Consumers use the registry resource or an explicit adapter; they do not define parallel resource models.
- Adding a type to the model does not imply that it is operationally supported.
- Future identity-format changes require a new version prefix and a reference-safe migration.

## Alternatives rejected

Provider names or display names alone are mutable and collide. Provider-specific core structs would duplicate policy and lifecycle behavior. Random IDs would avoid collisions but could not deterministically reconcile repeated discovery. Including owner, labels, environment, health, or size would create a new identity whenever ordinary operating data changed.
