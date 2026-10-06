// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"fmt"

	"github.com/lanl/conduit/defaults"
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
	nc := &NodesViperConfig{
		Nodes: make(map[string]*NViperConfig),
	}

	if err := viper.UnmarshalKey(defaults.ConfigNodesKey, &nc.Nodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node config: %w", err)
	}

	for name, node := range nc.Nodes {
		if node == nil {
			return nil, fmt.Errorf("node %q has an empty configuration", name)
		}

		if node.Address == "" {
			return nil, fmt.Errorf("node %q has no address configured", name)
		}

		if node.Port <= 0 {
			return nil, fmt.Errorf("node %q has invalid port %d", name, node.Port)
		}
	}

	return nc, nil
}
