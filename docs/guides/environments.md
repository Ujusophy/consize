# Environments

Consize lets you control the scope of infrastructure it can observe and modify.

This is useful when you want to start with a small environment and gradually expand automation.

## Control the scope

You can configure which Kubernetes namespaces Consize collects data from.

For example:

```yaml id="2e9q5m"
env:
  CONSIZE_NAMESPACES: boutique,payments,checkout
```

This limits collection to the specified namespaces.

To collect across the cluster:

```yaml id="9k3v7p"
env:
  CONSIZE_NAMESPACES: ""
```

## Read scope and write scope

Consize separates **what it can observe** from **what it can change**.

For example, you can allow Consize to:

* Read workloads across the cluster
* Analyze resources in multiple namespaces
* Apply changes only in `boutique`

This gives teams a way to use broad visibility while keeping modification permissions narrow.

## Production example

A production setup might look like:

```yaml id="6p2m8r"
collector:
  namespaces: []

rbac:
  writer:
    namespaces:
      - boutique
```

In this example:

* The collector can observe the cluster
* Direct changes are restricted to `boutique`

## Team-scoped example

For a smaller environment, you can restrict collection as well:

```yaml id="c7n4qx"
collector:
  namespaces:
    - boutique
    - checkout

rbac:
  writer:
    namespaces:
      - boutique
```

Now Consize only observes `boutique` and `checkout`, while direct changes are limited to `boutique`.

## Automatic application

Automatic application can be enabled for a specific namespace.

For example:

```sh id="p3k6vw"
kubectl label namespace boutique consize.savings.dev/auto-apply=enabled
```

This allows teams to explicitly mark which namespaces are eligible for automatic changes.

## Start small

A good way to introduce Consize is to begin with a limited scope:

```text id="x5j9ta"
One namespace
      ↓
Several workloads
      ↓
Recommendations
      ↓
Reviewable changes
      ↓
Limited automatic application
      ↓
Broader scope
```

This lets your team understand Consize's recommendations before expanding its permissions.

## Permissions matter

Namespace scope is only one part of the safety model.

Kubernetes RBAC determines what Consize can actually read or modify.

For more information, see [The Safety Net](../concepts/safety-net.md).

## Next steps

* [Production Installation](../getting-started/installation.md)
* [The Safety Net](../concepts/safety-net.md)
* [Configuration](../reference/configuration.md)
* [Kubernetes Rightsizing](rightsizing.md)
