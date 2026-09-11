# Configuration

Consize can be configured to control what it observes, what it can change, and how it operates in your environment.

This page covers the main configuration options.

## Collection scope

Use `CONSIZE_NAMESPACES` to control which Kubernetes namespaces Consize observes.

### Specific namespaces

```yaml id="8y4k2m"
env:
  CONSIZE_NAMESPACES: boutique,payments,checkout
```

### All namespaces

```yaml id="5t7n3q"
env:
  CONSIZE_NAMESPACES: ""
```

## Collector configuration

The collector can also be configured through Helm values.

For example:

```yaml id="3p6x9v"
collector:
  namespaces:
    - boutique
    - checkout
```

An empty list can be used when the collector should operate across the cluster:

```yaml id="1w8c5r"
collector:
  namespaces: []
```

## Writer configuration

Direct runtime changes can be restricted to specific namespaces.

For example:

```yaml id="6q2m7k"
rbac:
  writer:
    namespaces:
      - boutique
```

This allows the Consize writer to operate only within the configured namespace.

## Automatic application

A namespace can be explicitly marked as eligible for automatic application:

```sh id="4n7p2x"
kubectl label namespace boutique consize.savings.dev/auto-apply=enabled
```

This provides an additional boundary around where automated changes can occur.

## Metrics connection

Consize requires access to a Prometheus or compatible metrics endpoint.

The connection can be provided through the `consize-store` secret:

```sh id="9v3m6q"
kubectl -n consize-system create secret generic consize-store \
  --from-literal=prometheus-url='http://prometheus-operated.monitoring:9090'
```

Replace the URL with the metrics endpoint used by your environment.

## Notifications

Slack notifications can be configured through the `consize-alerts` secret:

```sh id="2k8r5n"
kubectl -n consize-system create secret generic consize-alerts \
  --from-literal=slack-webhook='https://hooks.slack.com/services/...'
```

## GitHub integration

GitHub integration can be configured using the `consize-github` secret:

```sh id="7m4q1p"
kubectl -n consize-system create secret generic consize-github \
  --from-literal=token='github_pat_...'
```

Keep GitHub credentials in Kubernetes secrets and avoid committing tokens to your repository.

## Configuration and permissions

Configuration controls what Consize is intended to do.

Kubernetes RBAC controls what Consize is actually allowed to do.

Both should be configured together.

For example, you might configure Consize to observe multiple namespaces while granting write access to only one:

```yaml id="5c9x2v"
collector:
  namespaces: []

rbac:
  writer:
    namespaces:
      - boutique
```

See [The Safety Net](../concepts/safety-net.md) for more information about permissions and guardrails.

## Helm configuration

Production installations use Helm values to configure Consize.

Example:

```sh id="3h6m9q"
helm upgrade --install consize ./charts/consize \
  --namespace consize-system \
  --create-namespace \
  -f ./charts/consize/examples/values-prod.yaml
```

Use your environment-specific values file to configure Consize for your deployment.

## Next steps

* [Production Installation](../getting-started/installation.md)
* [The Safety Net](../concepts/safety-net.md)
* [Environments](../guides/environments.md)
* [Kubernetes Rightsizing](../guides/rightsizing.md)
