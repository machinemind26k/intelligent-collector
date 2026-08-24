package runtimediscoveryextension

import "time"

type Config struct {
	Interval   time.Duration `mapstructure:"interval"`
	HostRoot   string        `mapstructure:"host_root"`
	StatusFile string        `mapstructure:"status_file"`
}
