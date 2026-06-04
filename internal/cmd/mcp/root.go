// Copyright 2026. Triad National Security, LLC. All rights reserved.

package mcpcmd

import (
	"fmt"
	"os"

	"github.com/lanl/conduit/defaults"
	"github.com/lanl/conduit/internal/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	debug   bool

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "conduit-mcp",
		Short: "start the conduit mcp server",
		Long:  `start the conduit mcp server`,
		Run: func(cmd *cobra.Command, args []string) {

			log := logger.NewConduitLogger(logrus.InfoLevel, "")
			if debug {
				log = logger.NewConduitLogger(logrus.DebugLevel, "")
			}

			log.Debug("hello world")

			os.Exit(0)
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		logrus.Errorf("failed to execute root command: %v", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() { initConfig(cfgFile) })

	// global flags
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is %s%s.%s)", DefaultConfigLocation, ConfigName, ConfigType))
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debugging")

	RootCmd.PersistentFlags().IntP("port", "p", DefaultPort, "Port to run conduit server on")
	RootCmd.PersistentFlags().StringSliceP("ip", "i", DefaultIPNet, "IP to run conduit server on")
	RootCmd.PersistentFlags().StringSlice("hostname", DefaultHostname, "The hostname for the conduit server. This is used for generating the tls cert")
	RootCmd.PersistentFlags().String("internal-ca-cert", DefaultInternalCACertLocation, "location of the internal ca cert .pem file")
	RootCmd.PersistentFlags().String("internal-ca-key", DefaultInternalCAKeyLocation, "location of the internal ca key .pem file")

	viper.BindPFlag(defaults.ConfigServerPortKey, RootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag(defaults.ConfigServerIPKey, RootCmd.PersistentFlags().Lookup("ip"))
	viper.BindPFlag(defaults.ConfigServerHostnameKey, RootCmd.PersistentFlags().Lookup("hostname"))
	viper.BindPFlag(defaults.ConfigInternalCACertKey, RootCmd.PersistentFlags().Lookup("internal-ca-cert"))
	viper.BindPFlag(defaults.ConfigInternalCAKeyKey, RootCmd.PersistentFlags().Lookup("internal-ca-key"))
}
