// Copyright 2026. Triad National Security, LLC. All rights reserved.

package servercmd

import (
	"net"

	"github.com/lanl/conduit/defaults"
	"github.com/lanl/conduit/internal/logger"
	"github.com/lanl/conduit/internal/pki"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	internalCACmd = &cobra.Command{
		Use:   "internal-ca",
		Short: "generate a CA for conduit internal use",
		Long:  `This subcommand generates a CA cert and key file for use internally within conduit`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.NewConduitLogger(logrus.InfoLevel, "")
			if debug {
				log = logger.NewConduitLogger(logrus.DebugLevel, "")
			}

			caCertPath := viper.GetString(defaults.ConfigInternalCACertKey)
			caKeyPath := viper.GetString(defaults.ConfigInternalCAKeyKey)
			_, err := pki.NewInternalCertManager(log, caCertPath, caKeyPath, nil, nil)
			if err != nil {
				log.Errorf("failed to create cert manager: %v", err)
				logrus.Exit(1)
			}
		},
	}
	externalCACmd = &cobra.Command{
		Use:   "external-ca",
		Short: "generate a CA for conduit external use",
		Long:  `This subcommand generates a CA cert and key file for use externally to communicate with conduit`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.NewConduitLogger(logrus.InfoLevel, "")
			if debug {
				log = logger.NewConduitLogger(logrus.DebugLevel, "")
			}

			caCertPath := viper.GetString(defaults.ConfigExternalCACertKey)
			caKeyPath := viper.GetString(defaults.ConfigExternalCAKeyKey)
			serverIPStrings := viper.GetStringSlice(defaults.ConfigServerIPKey)
			serverIPs := []net.IP{}
			for _, sips := range serverIPStrings {
				sip := net.ParseIP(sips)
				if sip == nil {
					log.Fatalf("failed to parse ip from string: %v", sips)
				}
				serverIPs = append(serverIPs, sip)
			}
			serverHostnames := viper.GetStringSlice(defaults.ConfigServerHostnameKey)
			_, err := pki.NewExternalCertManager(log, caCertPath, caKeyPath, serverIPs, serverHostnames)
			if err != nil {
				log.Errorf("failed to create cert manager: %v", err)
				logrus.Exit(1)
			}

		},
	}
)

func init() {
	RootCmd.AddCommand(internalCACmd)
	RootCmd.AddCommand(externalCACmd)
}
