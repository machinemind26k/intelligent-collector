package runtimediscoveryextension

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

var typeStr = component.MustNewType("runtime_discovery")

func NewFactory() extension.Factory {
	return extension.NewFactory(typeStr, createDefaultConfig, createExtension, component.StabilityLevelAlpha)
}

func createDefaultConfig() component.Config {
	return &Config{Interval: time.Minute, HostRoot: "/hostfs", StatusFile: "/var/lib/otelcol/runtime-capabilities.json"}
}

func createExtension(_ context.Context, settings extension.Settings, cfg component.Config) (extension.Extension, error) {
	return newExtension(cfg.(*Config), settings.Logger), nil
}
