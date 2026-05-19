// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"fmt"

	"github.com/spf13/viper"
)

type NodesViperConfig struct {
	Nodes map[string]*NViperConfig `mapstructure:"nodes" yaml:"nodes"`
}
type NViperConfig struct {
	Address   string `mapstructure:"address" yaml:"address"`
	Port      int    `mapstructure:"port" yaml:"port"`             // port of conduit-runner
	MinMemory string `mapstructure:"min-memory" yaml:"min-memory"` // minimum amount of available memory required to start a new job
	MaxJobs   int    `mapstructure:"max-jobs" yaml:"max-jobs"`     // maximum number of jobs allowed to run on the node concurrently
}

func GetNodeConfigsFromViper() (*NodesViperConfig, error) {
	nc := &NodesViperConfig{}
	err := viper.Unmarshal(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal etcd config: %v", err)
	}

	return nc, nil
}
