# Environments

Consize lets you control the scope of infrastructure it can observe and modify.

This is useful when you want to start with a small environment and gradually expand automation.

## Control the scope

You can configure which Kubernetes namespaces Consize collects data from.

For example:

```yaml
env:
  CONSIZE_NAMESPACES: boutique,payments,checkout
```

This limits collection to the specified namespaces.

To collect across the cluster:

```yaml
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

```yaml
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

```yaml
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

```sh
kubectl label namespace boutique consize.savings.dev/auto-apply=enabled
```

This allows teams to explicitly mark which namespaces are eligible for automatic changes.

## Start small

A good way to introduce Consize is to begin with a limited scope:

```text
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

Kubernetes RBAC determines what Consize can actually read or modify. The `rbac.writer.namespaces` values shown above are what the Helm chart uses to generate the actual `Role`/`RoleBinding` resources, see [Production Installation](../getting-started/installation.md#5-configure-kubernetes-permissions) for the underlying RBAC manifests if you need to configure permissions by hand instead of through Helm values.

For more information, see [The Safety Net](../concepts/safety-net.md).

## Next steps

* [Production Installation](../getting-started/installation.md)
* [The Safety Net](../concepts/safety-net.md)
* [Configuration](../reference/configuration.md)
* [Kubernetes Rightsizing](rightsizing.md)