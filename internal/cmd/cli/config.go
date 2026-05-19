// Copyright 2026. Triad National Security, LLC. All rights reserved.

package clicmd

import (
	"path/filepath"
	"strings"

	"github.com/lanl/conduit/defaults"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	defaultSystemConfigLocation = "/etc/conduit/"
	configName                  = "conduit-cli-config"
	configType                  = "yaml"
	envPrefix                   = "CONDUIT_CLI"
)

var (
	finalConfigPath = ""
)

// Find home directory.
func homeDir() string {
	home, err := homedir.Dir()
	if err != nil {
		logrus.Fatalf("Failed to get home directory: %v", err)
	}
	return home
}

func initConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		finalConfigPath = cfgFile

		viper.SetConfigName(strings.Split(filepath.Base(cfgFile), ".")[0])
		viper.SetConfigType(strings.TrimPrefix(filepath.Ext(cfgFile), "."))
		viper.AddConfigPath(filepath.Dir(cfgFile))
	} else {
		viper.SetConfigName(configName)
		viper.SetConfigType(configType)
		viper.AddConfigPath(defaultSystemConfigLocation)

		finalConfigPath = filepath.Join(defaultSystemConfigLocation, configName+"."+configType)
	}

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := viper.ReadInConfig(); err != nil {
		logrus.Errorf("failed to read config file: %v", err)
		logrus.Infof("creating default in %v", finalConfigPath)
		err := createDefaultConfig()
		if err != nil {
			logrus.Errorf("failed to create default config: %v", err)
		}
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	viper.SetEnvPrefix(envPrefix)

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	viper.AutomaticEnv()
}

func createDefaultConfig() error {
	viper.SetDefault(defaults.ConfigKrbConfigKey, defaults.DefaultKrb5Config)
	viper.SetDefault(defaults.ConfigKrbCacheKey, defaults.DefaultKrb5Cache)
	viper.SetDefault(defaults.ConfigKrbCachePrefixKey, defaults.DefaultKrb5CachePrefix)
	viper.SetDefault(defaults.ConfigKrbSpnKey, defaults.DefaultSPN)
	viper.SetDefault(defaults.ConfigKrbKinitPathKey, defaults.DefaultKinitPath)

	viper.SetDefault(defaults.ConfigConduitIPKey, defaults.DefaultConduitHost)
	viper.SetDefault(defaults.ConfigConduitPortKey, defaults.DefaultConduitPort)
	viper.SetDefault(defaults.ConfigConduitCAKey, defaults.DefaultConduitCA)
	viper.SetDefault(defaults.ConfigConduitTimeoutKey, defaults.DefaultReqTimeout)

	viper.SetDefault(defaults.ConfigClientGrpcLimitKey, defaults.DefaultClientGRPCLimit)
	viper.SetDefault(defaults.ConfigClientCertKey, defaults.DefaultClientCert)
	viper.SetDefault(defaults.ConfigClientKeyKey, defaults.DefaultClientKey)

	return viper.SafeWriteConfig()
}
