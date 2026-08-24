# Install and run Intelligent Collector

## 1. Prerequisites

- Kubernetes cluster with permission to create a namespace, ClusterRole, ClusterRoleBinding, ConfigMaps, Secrets, DaemonSets, Deployments, and Services.
- `kubectl` configured for that cluster.
- Docker or another OCI-compatible builder, plus access to an image registry reachable by every cluster node.
- A Prometheus remote-write endpoint and a Loki OTLP endpoint, or replacements configured in the Gateway manifest.
- A TLS PKI that can issue one Gateway server certificate and one client certificate per node agent identity.

The node agent runs with host access to collect host, kubelet, and Docker data. Review the security model in [Architecture and LLD](NODE_AGENT_COMPLETE_ARCHITECTURE_LLD.md) before deploying it to production.

## 2. Configure values

Edit [deploy/otel/kubernetes.yaml](deploy/otel/kubernetes.yaml) before deployment.

Replace these placeholders:

| Location | Replace with |
|---|---|
| DaemonSet `image` | Your published image, for example `registry.example.com/observability/intelligent-collector:0.1.0` |
| `ACCOUNT_ID`, `SITE_ID` | Your tenant/account scope |
| `K8S_CLUSTER_NAME` | A stable cluster name |
| `OTEL_GATEWAY_ENDPOINT` | Gateway DNS name and port if different from the included Service |
| Gateway `PROMETHEUS_REMOTE_WRITE_ENDPOINT` | Your Prometheus-compatible remote-write URL |
| Gateway `LOKI_OTLP_ENDPOINT` | Your Loki OTLP URL |

Do not put private keys, tokens, or tenant credentials in Pod annotations or Docker labels.

## 3. Build and publish the Collector image

From the repository root:

```sh
export IMAGE=registry.example.com/observability/intelligent-collector:0.1.0
docker build -t "$IMAGE" .
docker push "$IMAGE"
```

Update the DaemonSet image in `deploy/otel/kubernetes.yaml` to exactly the same value. The image build uses the pinned OpenTelemetry Collector Builder and includes the custom `runtime_discovery` extension.

## 4. Create the namespace and ConfigMaps

```sh
kubectl apply -f deploy/otel/kubernetes.yaml --server-side --dry-run=server

kubectl create namespace observability --dry-run=client -o yaml | kubectl apply -f -

kubectl -n observability create configmap otel-node-agent-config \
  --from-file=config.yaml=deploy/otel/agent-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n observability create configmap otel-gateway-config \
  --from-file=config.yaml=deploy/otel/gateway-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -
```

The server-side dry run validates permissions and Kubernetes object schema without creating resources. It will fail until the TLS Secrets in the next section are also present only if your admission policy requires them at validation time.

## 5. Create TLS Secrets

Use certificates issued by your organization’s CA. The Gateway server certificate must be valid for `otel-gateway.observability.svc` (and any other name agents use). The agent certificate must be signed by the CA configured as the Gateway’s client CA.

```sh
# Node agent: CA that validates the Gateway + agent client certificate/key.
kubectl -n observability create secret generic otel-node-agent-tls \
  --from-file=ca.crt=/secure/path/gateway-ca.crt \
  --from-file=tls.crt=/secure/path/agent-client.crt \
  --from-file=tls.key=/secure/path/agent-client.key \
  --dry-run=client -o yaml | kubectl apply -f -

# Gateway: server certificate/key + CA that validates agent client certificates.
kubectl -n observability create secret generic otel-gateway-tls \
  --from-file=tls.crt=/secure/path/gateway-server.crt \
  --from-file=tls.key=/secure/path/gateway-server.key \
  --from-file=client-ca.crt=/secure/path/agent-client-ca.crt \
  --dry-run=client -o yaml | kubectl apply -f -
```

For production, use your certificate-management system to rotate Secrets and roll/reload the workloads in line with its policy. Never commit certificate material into this repository.

## 6. Install

```sh
kubectl apply -f deploy/otel/kubernetes.yaml
kubectl -n observability rollout status deployment/otel-gateway --timeout=5m
kubectl -n observability rollout status daemonset/otel-node-agent --timeout=10m
```

The DaemonSet creates one node agent per schedulable node. The Gateway Deployment has two replicas. If your cluster does not run Docker, Docker capability diagnostics will report `unavailable`; host and Kubernetes collection continue.

## 7. Verify operation

```sh
kubectl -n observability get pods -o wide
kubectl -n observability get daemonset otel-node-agent
kubectl -n observability logs daemonset/otel-node-agent --tail=100
kubectl -n observability logs deployment/otel-gateway --tail=100
```

Verify that every desired node has an `otel-node-agent` Pod in `Running` state. Check the Gateway logs for successful OTLP exports and your Prometheus/Loki backends for host and workload data.

To inspect capability discovery on one node:

```sh
AGENT_POD=$(kubectl -n observability get pods -l app.kubernetes.io/name=otel-node-agent -o jsonpath='{.items[0].metadata.name}')
kubectl -n observability exec "$AGENT_POD" -- cat /var/lib/otelcol/runtime-capabilities.json
```

The file reports `available`, `blocked`, or `unavailable` for host, Docker, Podman, Kubernetes, containerd, CRI-O, and KVM/libvirt probes.

## 8. Enable dynamic workload collection

Dynamic Prometheus scraping is intentionally opt-in. Add this Pod annotation to a workload with a declared metrics port:

```yaml
metadata:
  annotations:
    observability.opentelemetry.io/scrape: "true"
```

The agent starts a receiver for that Pod’s `/metrics` endpoint and removes it when the Pod/port/annotation disappears.

To collect a specific Kubernetes container’s logs dynamically:

```yaml
metadata:
  annotations:
    io.opentelemetry.discovery.logs.app/enabled: "true"
```

Replace `app` with the exact container name. Docker application endpoints use the equivalent Docker label:

```text
observability.opentelemetry.io/scrape=true
```

They must expose a published port reachable from the node-agent Pod.

## 9. Upgrade and uninstall

For a configuration update, recreate the appropriate ConfigMap and restart the affected workload:

```sh
kubectl -n observability create configmap otel-node-agent-config \
  --from-file=config.yaml=deploy/otel/agent-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n observability rollout restart daemonset/otel-node-agent
```

To remove the workloads while retaining the namespace and Secrets for investigation:

```sh
kubectl delete -f deploy/otel/kubernetes.yaml
```

Delete ConfigMaps, Secrets, and the `observability` namespace separately only when you no longer need their data.
