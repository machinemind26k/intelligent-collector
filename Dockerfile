FROM golang:1.25-bookworm AS builder
WORKDIR /src
COPY collector-builder.yaml ./
COPY custom-components ./custom-components
RUN mkdir -p _build && \
    go install go.opentelemetry.io/collector/cmd/builder@v0.158.0 && \
    $(go env GOPATH)/bin/builder --config=collector-builder.yaml

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /src/_build/observability-collector/observability-collector /otelcol
COPY deploy/otel/agent-config.yaml /etc/otelcol/config.yaml
USER 65532:65532
ENTRYPOINT ["/otelcol"]
CMD ["--config=/etc/otelcol/config.yaml"]
