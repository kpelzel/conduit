// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"fmt"
	"net"

	"github.com/lanl/conduit/defaults"
	"github.com/spf13/viper"
)

type EtcdViperConfig struct {
	Etcd []*EViperConfig `mapstructure:"etcd" yaml:"etcd"`
}
type EViperConfig struct {
	Hostname string `mapstructure:"hostname" yaml:"hostname"`
	IP       string `mapstructure:"ip" yaml:"ip"`
	Port     int    `mapstructure:"port" yaml:"port"`
}

func GetEtcdEndpointsFromViper() ([]string, error) {
	var configs []*EViperConfig

	if err := viper.UnmarshalKey(defaults.ConfigETCDKey, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal etcd config: %w", err)
	}

	var etcdEndpoints []string

	for _, e := range configs {
		ip := net.ParseIP(e.IP)
		etcdEndpoints = append(
			etcdEndpoints,
			fmt.Sprintf("%s:%d", ip.String(), e.Port),
		)
	}

	return etcdEndpoints, nil
}
