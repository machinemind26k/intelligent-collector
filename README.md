# Intelligent Collector

Intelligent Collector is a Collector-first node observability agent built as a custom OpenTelemetry Collector Contrib distribution. It discovers node-local Docker and Kubernetes workloads, dynamically creates/removes opt-in workload receivers, collects host and runtime telemetry, and exports OTLP to a Gateway over mTLS.

```text
Intelligent Collector DaemonSet -> OTLP/gRPC + mTLS -> Collector Gateway -> Prometheus remote write + Loki OTLP -> Grafana
```

## What it does

- Collects host CPU, memory, disk, filesystem, network, paging, and process metrics.
- Collects Docker container CPU, memory, network, and block-I/O metrics.
- Collects Kubernetes node, Pod, container, and volume metrics through the kubelet.
- Dynamically starts/stops Prometheus and file-log child receivers for opted-in Docker/Kubernetes workloads.
- Enriches telemetry with trusted tenant, agent, host, cluster, and workload metadata.
- Applies memory limits, batching, persistent queues, retry, and mTLS-protected OTLP export.

## Documentation

Read these documents in order:

| Document | Description |
|---|---|
| [Architecture and LLD](docs/NODE_AGENT_COMPLETE_ARCHITECTURE_LLD.md) | HLD, LLD, component model, discovery/activation sequences, telemetry inventory, security, and failure isolation |
| [Install and run](docs/INSTALL_AND_RUN.md) | Prerequisites, image build, mTLS Secrets, Kubernetes installation, verification, dynamic workload opt-in, upgrade, and uninstall |
| [Deployment reference](docs/DEPLOYMENT.md) | Runtime receiver lifecycle, Collector ConfigMap commands, and annotation examples |
| [Node Collector configuration](deploy/otel/agent-config.yaml) | Receivers, dynamic receiver rules, processors, queue, retry, and OTLP/mTLS export |
| [Gateway configuration](deploy/otel/gateway-config.yaml) | mTLS receiver and backend exports |
| [Kubernetes manifests](deploy/otel/kubernetes.yaml) | RBAC, DaemonSet, Service, and Gateway deployment |

## Repository layout

```text
docs/                               All architecture, installation, and deployment documentation
deploy/otel/                        Collector configuration and Kubernetes manifests
custom-components/runtime-discoveryextension/  Runtime capability diagnostic extension
collector-builder.yaml              Custom Collector component registration
Dockerfile                          Custom Collector image build
```

## Start here

Follow [Install and run](docs/INSTALL_AND_RUN.md). In summary:

```sh
export IMAGE=registry.example.com/observability/intelligent-collector:0.1.0
docker build -t "$IMAGE" .
docker push "$IMAGE"
```

Update the image and environment values in `deploy/otel/kubernetes.yaml`, create the mTLS Secrets and ConfigMaps described in the installation guide, then apply the manifests.
