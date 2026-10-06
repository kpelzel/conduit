// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"fmt"
	"net"
	"strconv"

	"github.com/lanl/conduit/defaults"
	"github.com/spf13/viper"
)

type RqlitePath string

const (
	PathExecute RqlitePath = "/db/execute"
	PathQuery   RqlitePath = "/db/query"
)

type RqliteViperConfig struct {
	Rqlite []*RViperConfig `mapstructure:"rqlite" yaml:"rqlite"`
}

type RViperConfig struct {
	Hostname string `mapstructure:"hostname" yaml:"hostname"`
	IP       string `mapstructure:"ip" yaml:"ip"`
	Port     int    `mapstructure:"port" yaml:"port"`
}

func GetRqliteEndpointsFromViper() ([]string, error) {
	var configs []*RViperConfig

	if err := viper.UnmarshalKey(defaults.ConfigRqliteKey, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rqlite config: %w", err)
	}

	var rqliteEndpoints []string

	for _, r := range configs {
		ip := net.ParseIP(r.IP)
		addr := net.JoinHostPort(ip.String(), strconv.Itoa(r.Port))
		rqliteEndpoints = append(rqliteEndpoints, addr)
	}

	return rqliteEndpoints, nil
}
