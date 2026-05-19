// Copyright 2026. Triad National Security, LLC. All rights reserved.

package servercmd

import (
	"net"
	"path/filepath"
	"time"

	"github.com/lanl/conduit/defaults"
	"github.com/lanl/conduit/internal/logger"
	"github.com/lanl/conduit/internal/pki"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	externalClientCertCmd = &cobra.Command{
		Use:   "external-client-cert",
		Short: "generate client cert and key for external communication with conduit",
		Long:  `This subcommand generates a mTLS client cert and key using the CA for external communication with conduit`,
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
			cm, err := pki.NewExternalCertManager(log, caCertPath, caKeyPath, serverIPs, serverHostnames)
			if err != nil {
				log.Fatalf("failed to create cert manager: %v", err)
			}

			outputPath, err := cmd.Flags().GetString("output")
			if err != nil {
				log.Fatalf("failed to get output flag: %v", err)
			}
			certPath := outputPath
			keyPath := outputPath

			if cmd.Flags().Lookup("separate-cert-key").Changed {
				certName, err := cmd.Flags().GetString("cert-name")
				if err != nil {
					log.Fatalf("failed to get cert-name flag: %v", err)
				}
				keyName, err := cmd.Flags().GetString("key-name")
				if err != nil {
					log.Fatalf("failed to get key-name flag: %v", err)
				}

				certPath = filepath.Join(filepath.Dir(outputPath), certName)
				keyPath = filepath.Join(filepath.Dir(outputPath), keyName)
			}

			clientCN := "conduit-external-client"
			if cmd.Flags().Lookup("client-commonname").Changed {
				clientCN, err = cmd.Flags().GetString("client-commonname")
				if err != nil {
					log.Fatalf("failed to get client-commonname flag: %v", err)
				}
			}

			// get expiration from user
			expiration := time.Now().AddDate(0, 0, DefaultClientCertExpirationDays)
			if cmd.Flags().Lookup("expiration-date").Changed && cmd.Flags().Lookup("expiration").Changed {
				log.Fatalf("cannot specify both expiration-date and expiration flags together")
			} else if cmd.Flags().Lookup("expiration-date").Changed {
				providedDate, err := cmd.Flags().GetString("expiration-date")
				if err != nil {
					log.Fatalf("failed to get expiration-date flag: %v", err)
				}
				expiration, err = time.ParseInLocation(time.RFC3339, providedDate, time.Local)
				if err != nil {
					log.Fatalf("failed to parse provided date in expiration-date flag: %v", err)
				}
			} else if cmd.Flags().Lookup("expiration").Changed {
				providedDays, err := cmd.Flags().GetInt("expiration")
				if err != nil {
					log.Fatalf("failed to get expiration flag: %v", err)
				}
				expiration = time.Now().AddDate(0, 0, providedDays)
			}

			err = cm.WriteClientCredsToFile(certPath, keyPath, clientCN, expiration)
			if err != nil {
				log.Fatalf("failed to write client cert and key to file: %v", err)
			}
		},
	}

	internalClientCertCmd = &cobra.Command{
		Use:   "internal-client-cert",
		Short: "generate client cert and key for internal communication with etcd or rqlite",
		Long:  `This subcommand generates a mTLS client cert and key using the CA for internal communication with etcd or rqlite`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.NewConduitLogger(logrus.InfoLevel, "")
			if debug {
				log = logger.NewConduitLogger(logrus.DebugLevel, "")
			}

			caCertPath := viper.GetString(defaults.ConfigInternalCACertKey)
			caKeyPath := viper.GetString(defaults.ConfigInternalCAKeyKey)
			cm, err := pki.NewInternalCertManager(log, caCertPath, caKeyPath, nil, nil)
			if err != nil {
				log.Fatalf("failed to create cert manager: %v", err)
			}

			outputPath, err := cmd.Flags().GetString("output")
			if err != nil {
				log.Fatalf("failed to get output flag: %v", err)
			}
			certPath := outputPath
			keyPath := outputPath

			if cmd.Flags().Lookup("separate-cert-key").Changed {
				certName, err := cmd.Flags().GetString("cert-name")
				if err != nil {
					log.Fatalf("failed to get cert-name flag: %v", err)
				}
				keyName, err := cmd.Flags().GetString("key-name")
				if err != nil {
					log.Fatalf("failed to get key-name flag: %v", err)
				}

				certPath = filepath.Join(filepath.Dir(outputPath), certName)
				keyPath = filepath.Join(filepath.Dir(outputPath), keyName)
			}

			clientCN := "conduit-internal-client"
			if cmd.Flags().Lookup("client-commonname").Changed {
				clientCN, err = cmd.Flags().GetString("client-commonname")
				if err != nil {
					log.Fatalf("failed to get client-commonname flag: %v", err)
				}
			}

			// get expiration from user
			expiration := time.Now().AddDate(0, 0, DefaultClientCertExpirationDays)
			if cmd.Flags().Lookup("expiration-date").Changed && cmd.Flags().Lookup("expiration").Changed {
				log.Fatalf("cannot specify both expiration-date and expiration flags together")
			} else if cmd.Flags().Lookup("expiration-date").Changed {
				providedDate, err := cmd.Flags().GetString("expiration-date")
				if err != nil {
					log.Fatalf("failed to get expiration-date flag: %v", err)
				}
				expiration, err = time.ParseInLocation(time.RFC3339, providedDate, time.Local)
				if err != nil {
					log.Fatalf("failed to parse provided date in expiration-date flag: %v", err)
				}
			} else if cmd.Flags().Lookup("expiration").Changed {
				providedDays, err := cmd.Flags().GetInt("expiration")
				if err != nil {
					log.Fatalf("failed to get expiration flag: %v", err)
				}
				expiration = time.Now().AddDate(0, 0, providedDays)
			}

			err = cm.WriteClientCredsToFile(certPath, keyPath, clientCN, expiration)
			if err != nil {
				log.Fatalf("failed to write client cert and key to file: %v", err)
			}
		},
	}

	internalServerCertCmd = &cobra.Command{
		Use:   "internal-server-cert",
		Short: "generate server cert and key for use internally within conduit",
		Long:  `This subcommand generates a TLS server cert and key using the CA for internal conduit communication`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.NewConduitLogger(logrus.InfoLevel, "")
			if debug {
				log = logger.NewConduitLogger(logrus.DebugLevel, "")
			}

			caCertPath := viper.GetString(defaults.ConfigInternalCACertKey)
			caKeyPath := viper.GetString(defaults.ConfigInternalCAKeyKey)
			cm, err := pki.NewInternalCertManager(log, caCertPath, caKeyPath, nil, nil)
			if err != nil {
				log.Fatalf("failed to create cert manager: %v", err)
			}

			outputPath, err := cmd.Flags().GetString("output")
			if err != nil {
				log.Fatalf("failed to get output flag: %v", err)
			}
			certPath := outputPath
			keyPath := outputPath

			if cmd.Flags().Lookup("separate-cert-key").Changed {
				certName, err := cmd.Flags().GetString("cert-name")
				if err != nil {
					log.Fatalf("failed to get cert-name flag: %v", err)
				}
				keyName, err := cmd.Flags().GetString("key-name")
				if err != nil {
					log.Fatalf("failed to get key-name flag: %v", err)
				}

				certPath = filepath.Join(filepath.Dir(outputPath), certName)
				keyPath = filepath.Join(filepath.Dir(outputPath), keyName)
			}

			var newServerHostnames []string
			if cmd.Flags().Lookup("server-hostname").Changed {
				newServerHostnames, err = cmd.Flags().GetStringSlice("server-hostname")
				if err != nil {
					log.Fatalf("failed to get server-hostname flag: %v", err)
				}
			}

			var newServerIPs []net.IP
			if cmd.Flags().Lookup("server-ip").Changed {
				newServerIPs, err = cmd.Flags().GetIPSlice("server-ip")
				if err != nil {
					log.Fatalf("failed to get server-ip flag: %v", err)
				}
			}

			serverCN := "conduit-internal-server"
			if cmd.Flags().Lookup("server-commonname").Changed {
				serverCN, err = cmd.Flags().GetString("server-commonname")
				if err != nil {
					log.Fatalf("failed to get server-commonname flag: %v", err)
				}
			}

			err = cm.WriteServerCredsToFile(certPath, keyPath, newServerHostnames, newServerIPs, serverCN)
			if err != nil {
				log.Fatalf("failed to create server key and cert: %v", err)
			}
		},
	}
)

func init() {
	externalClientCertCmd.Flags().CountP("separate-cert-key", "s", "separate the client key and cert into separate pem files")
	externalClientCertCmd.Flags().String("cert-name", "conduit_client_cert.pem", "the output path of for the key and cert. (only used when --separate-cert-key is specified)")
	externalClientCertCmd.Flags().String("key-name", "conduit_client_key.pem", "the output path of for the key and cert. (only used when --separate-cert-key is specified)")
	externalClientCertCmd.Flags().StringP("output", "o", "./conduit_client.pem", "the output path of for the key and cert")

	externalClientCertCmd.Flags().String("client-commonname", "conduit-external-client", "the commonname for the _client_ that will be included in the client cert")
	externalClientCertCmd.Flags().IntP("expiration", "e", DefaultClientCertExpirationDays, "the number of days from creation that the client cert will expire")
	externalClientCertCmd.Flags().String("expiration-date", time.Now().AddDate(0, 0, DefaultClientCertExpirationDays).Local().Format(time.RFC3339), "a date (in RFC3339 format) that the client cert will expire. Will use system locale if none is provided")

	internalClientCertCmd.Flags().CountP("separate-cert-key", "s", "separate the client key and cert into separate pem files")
	internalClientCertCmd.Flags().String("cert-name", "conduit_client_cert.pem", "the output path of for the key and cert. (only used when --separate-cert-key is specified)")
	internalClientCertCmd.Flags().String("key-name", "conduit_client_key.pem", "the output path of for the key and cert. (only used when --separate-cert-key is specified)")
	internalClientCertCmd.Flags().StringP("output", "o", "./conduit_client.pem", "the output path of for the key and cert")

	internalClientCertCmd.Flags().String("client-commonname", "conduit-internal-client", "the commonname for the _client_ that will be included in the client cert")
	internalClientCertCmd.Flags().IntP("expiration", "e", DefaultClientCertExpirationDays, "the number of days from creation that the client cert will expire")
	internalClientCertCmd.Flags().String("expiration-date", time.Now().AddDate(0, 0, DefaultClientCertExpirationDays).Local().Format(time.RFC3339), "a date (in RFC3339 format) that the client cert will expire. Will use system locale if none is provided")

	internalServerCertCmd.Flags().CountP("separate-cert-key", "s", "separate the server key and cert into separate pem files")
	internalServerCertCmd.Flags().String("cert-name", "conduit_server_cert.pem", "the output path of for the key and cert. (only used when --separate-cert-key is specified)")
	internalServerCertCmd.Flags().String("key-name", "conduit_server_key.pem", "the output path of for the key and cert. (only used when --separate-cert-key is specified)")
	internalServerCertCmd.Flags().StringP("output", "o", "./conduit_server.pem", "the output path of for the key and cert")

	internalServerCertCmd.Flags().String("server-commonname", "conduit-internal-server", "the commonname for the _server_ that will be included in the server cert")
	internalServerCertCmd.Flags().IPSlice("server-ip", nil, "the ip(s) of the server that will be included in the server cert")
	internalServerCertCmd.Flags().StringSlice("server-hostname", nil, "the hostname(s) of the server that will be included in the server cert")
	// serverCertCmd.MarkFlagRequired("hostname")

	RootCmd.AddCommand(externalClientCertCmd)
	RootCmd.AddCommand(internalClientCertCmd)
	RootCmd.AddCommand(internalServerCertCmd)
}
