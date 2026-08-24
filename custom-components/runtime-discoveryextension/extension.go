package runtimediscoveryextension

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type Capability struct {
	Name, State, Reason, Endpoint string    `json:"name"`
	ObservedAt                    time.Time `json:"observed_at"`
}

type extensionImpl struct {
	cfg    *Config
	logger *zap.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newExtension(cfg *Config, logger *zap.Logger) *extensionImpl {
	return &extensionImpl{cfg: cfg, logger: logger}
}
func (e *extensionImpl) Start(ctx context.Context, _ component.Host) error {
	runCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.wg.Add(1)
	go func() { defer e.wg.Done(); e.loop(runCtx) }()
	return nil
}
func (e *extensionImpl) Shutdown(context.Context) error {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
	return nil
}
func (e *extensionImpl) loop(ctx context.Context) {
	every := e.cfg.Interval
	if every <= 0 {
		every = time.Minute
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		e.writeStatus()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (e *extensionImpl) writeStatus() {
	caps := discover(e.cfg.HostRoot)
	b, err := json.MarshalIndent(caps, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(e.cfg.StatusFile), 0700); err == nil {
		if err = os.WriteFile(e.cfg.StatusFile, b, 0600); err != nil {
			e.logger.Warn("cannot write runtime capability status", zap.Error(err))
		}
	}
}
func discover(root string) []Capability {
	now := time.Now().UTC()
	caps := []Capability{pathCapability("host", filepath.Join(root, "proc/stat"), now), socketCapability("docker", filepath.Join(root, "var/run/docker.sock"), now), socketCapability("containerd", filepath.Join(root, "run/containerd/containerd.sock"), now), socketCapability("cri-o", filepath.Join(root, "var/run/crio/crio.sock"), now), pathCapability("kubernetes", filepath.Join(root, "var/lib/kubelet"), now), kvmCapability(root, now)}
	podman := []string{filepath.Join(root, "run/podman/podman.sock")}
	if users, err := os.ReadDir(filepath.Join(root, "run/user")); err == nil {
		for _, u := range users {
			podman = append(podman, filepath.Join(root, "run/user", u.Name(), "podman/podman.sock"))
		}
	}
	for _, p := range podman {
		c := socketCapability("podman", p, now)
		if c.State != "unavailable" {
			caps = append(caps, c)
			break
		}
	}
	sort.Slice(caps, func(i, j int) bool { return caps[i].Name < caps[j].Name })
	return caps
}
func pathCapability(name, path string, at time.Time) Capability {
	if _, err := os.Stat(path); err == nil {
		return Capability{Name: name, State: "available", Reason: "local path is present", Endpoint: path, ObservedAt: at}
	}
	return Capability{Name: name, State: "unavailable", Reason: "local path is absent", ObservedAt: at}
}
func socketCapability(name, path string, at time.Time) Capability {
	if _, err := os.Stat(path); err != nil {
		return Capability{Name: name, State: "unavailable", Reason: "runtime socket is absent", ObservedAt: at}
	}
	c, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err != nil {
		return Capability{Name: name, State: "blocked", Reason: "runtime socket is not accessible", Endpoint: path, ObservedAt: at}
	}
	_ = c.Close()
	return Capability{Name: name, State: "available", Reason: "runtime socket is accessible", Endpoint: path, ObservedAt: at}
}
func kvmCapability(root string, at time.Time) Capability {
	for _, p := range []string{filepath.Join(root, "var/run/libvirt/libvirt-sock"), filepath.Join(root, "dev/kvm")} {
		if _, err := os.Stat(p); err == nil {
			return Capability{Name: "kvm", State: "available", Reason: "libvirt or KVM is present", Endpoint: p, ObservedAt: at}
		}
	}
	return Capability{Name: "kvm", State: "unavailable", Reason: "libvirt socket and KVM device are absent", ObservedAt: at}
}
