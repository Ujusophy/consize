````markdown
# Production Installation

This guide walks you through installing Consize on a Kubernetes cluster using Helm.

## Prerequisites

Before installing Consize, make sure you have:

- A running Kubernetes cluster
- `kubectl` configured for the cluster
- Helm installed
- Prometheus or a compatible metrics endpoint

## 1. Create the Consize namespace

```sh
kubectl create namespace consize-system
````

## 2. Configure your metrics connection

Consize needs access to your Prometheus metrics endpoint.

Create the required secret:

```sh
kubectl -n consize-system create secret generic consize-store \
  --from-literal=prometheus-url='http://prometheus-operated.monitoring:9090'
```

Replace the URL with your Prometheus endpoint if it is different.

## 3. Configure optional integrations

### Slack

To enable Slack notifications:

```sh
kubectl -n consize-system create secret generic consize-alerts \
  --from-literal=slack-webhook='https://hooks.slack.com/services/...'
```

### GitHub

To enable GitHub integration:

```sh
kubectl -n consize-system create secret generic consize-github \
  --from-literal=token='github_pat_...'
```

### Cloud provider credentials

Consize can also use cloud provider credentials when required.

See the configuration reference for provider-specific settings.

## 4. Configure collection scope

By default, Consize can collect data across the cluster.

To limit collection to specific namespaces:

```yaml
env:
  CONSIZE_NAMESPACES: boutique,payments,checkout
```

To collect across all namespaces:

```yaml
env:
  CONSIZE_NAMESPACES: ""
```

## 5. Configure Kubernetes permissions

Consize uses Kubernetes RBAC to control what it can read and what it can change.

### Read-only access

The collector requires read access to the resources it analyzes.

Example:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: consize-reader
  namespace: consize-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: consize-read
rules:
  - apiGroups: [""]
    resources: ["namespaces", "pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["batch"]
    resources: ["jobs", "cronjobs"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: consize-read
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: consize-read
subjects:
  - kind: ServiceAccount
    name: consize-reader
    namespace: consize-system
```

### Direct apply access

If you want Consize to apply approved changes directly, grant write access only to the namespaces where it should operate.

For example:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: consize-writer
  namespace: consize-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: consize-apply
  namespace: boutique
rules:
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: consize-apply
  namespace: boutique
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: consize-apply
subjects:
  - kind: ServiceAccount
    name: consize-writer
    namespace: consize-system
```

You can then enable automatic application for a namespace:

```sh
kubectl label namespace boutique consize.savings.dev/auto-apply=enabled
```

## 6. Install Consize with Helm

From the Consize repository:

```sh
helm upgrade --install consize ./charts/consize \
  --namespace consize-system \
  --create-namespace \
  -f ./charts/consize/examples/values-prod.yaml
```

For more advanced deployments, configure the Helm values for your environment.

## 7. Verify the installation

Check the Consize pods:

```sh
kubectl -n consize-system get pods
```

Check the scheduled jobs:

```sh
kubectl -n consize-system get cronjobs
```

Then check the API health endpoint:

```sh
kubectl -n consize-system port-forward svc/consize-api 18099:8080
```

In another terminal:

```sh
curl http://127.0.0.1:18099/readyz
```

Expected response:

```json
{"status":"ready"}
```

## Next steps

* [Quickstart](quickstart.md)
* [Interactive Sandbox](sandbox.md)
* [Configuration](../reference/configuration.md)
* [How Consize Works](../concepts/architecture.md)
* [The Safety Net](../concepts/safety-net.md)

````

### Then do this

Save the file and run:

```bash
mkdocs serve
````