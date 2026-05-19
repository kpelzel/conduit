// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"fmt"
	"net"

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
	var etcdEndpoints []string

	ec := &EtcdViperConfig{}
	err := viper.Unmarshal(ec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal etcd config: %v", err)
	}
	for _, e := range ec.Etcd {
		ip := net.ParseIP(e.IP)
		etcdEndpoints = append(etcdEndpoints, fmt.Sprintf("%s:%d", ip.String(), e.Port))
	}

	return etcdEndpoints, nil
}
