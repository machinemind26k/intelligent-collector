# Collector-first deployment

Use `otel/opentelemetry-collector-contrib` for the node agent and Gateway Collector. The Contrib distribution provides `hostmetrics`, `docker_stats`, `kubeletstats`, `filelog`, and the processors/exporters configured here.

The node Collector runs as a DaemonSet. It collects host metrics on its local node, Docker metrics through the mounted Docker socket, kubelet pod/container metrics, and explicitly opted-in container logs; it then sends OTLP/gRPC to the Gateway using mTLS.

## Runtime receiver lifecycle

The agent does not restart to discover a workload. `docker_observer` watches Docker container events and `k8s_observer` watches Pods scheduled on its node. Each observer sends endpoint-added and endpoint-removed events to `receiver_creator`; it starts a matching receiver on add and shuts that receiver down on remove.

Two dynamic paths are enabled:

- A Docker container with a published metrics port and label `observability.opentelemetry.io/scrape=true` gets an individual Prometheus receiver. That receiver ends when the container stops or the label/port no longer matches.
- A Kubernetes Pod container with `io.opentelemetry.discovery.logs.<container-name>/enabled: "true"` gets its own file-log receiver. It is stopped when the container disappears. A Pod with `observability.opentelemetry.io/scrape: "true"` receives an individual Prometheus receiver for each declared container port.

The dynamically discovered HTTP metrics endpoint is intentionally opt-in and defaults to `/metrics`. Docker resource metrics (`docker_stats`), host metrics, and kubelet metrics are node-level collectors, so they remain long-lived receivers rather than one receiver per workload.

Example Kubernetes annotations:

```yaml
metadata:
  annotations:
    observability.opentelemetry.io/scrape: "true"
    io.opentelemetry.discovery.logs.app/enabled: "true"
```

The Gateway requires client certificates and routes metrics to a Prometheus remote-write endpoint and logs to Loki's native OTLP endpoint.

Create the ConfigMaps from the versioned configuration files before applying the workload manifest:

```sh
kubectl -n observability create configmap otel-node-agent-config \
  --from-file=config.yaml=deploy/otel/agent-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n observability create configmap otel-gateway-config \
  --from-file=config.yaml=deploy/otel/gateway-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deploy/otel/kubernetes.yaml
```

The repository contains no legacy custom Go agent. `custom-components/runtime-discoveryextension` is a Collector extension built into the custom Collector image by the OpenTelemetry Collector Builder. It checks runtime sockets/read-only paths every 60 seconds and writes capability state to the persistent Collector storage directory. CRI/containerd and libvirt/KVM require Collector receiver components (custom distribution if no supported upstream receiver exists), not a separate agent process.

Before deployment, set the environment variables used by the configs and provide TLS Secrets. Validate against the exact Collector Contrib image version that you pin; component availability and configuration schemas are release-specific.
