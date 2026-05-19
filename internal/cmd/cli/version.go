// Copyright 2026. Triad National Security, LLC. All rights reserved.

package clicmd

import (
	"fmt"
	"os"

	rDebug "runtime/debug"

	"github.com/lanl/conduit/defaults"
	"github.com/lanl/conduit/internal/cli/client"
	"github.com/lanl/conduit/internal/cli/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print cli version and conduit server version",
	Long:  `print cli version and conduit server version`,
	Run: func(cmd *cobra.Command, args []string) {
		version := ""
		modified := false
		if info, ok := rDebug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					version = setting.Value
				}
				if setting.Key == "vcs.modified" {
					if setting.Value == "true" {
						modified = true
					}
				}
			}
		}

		if modified {
			version += " (modified)"
		}

		fmt.Printf("conduit-cli version: %s\n", version)

		logger := logrus.New()
		if debug {
			logger.SetLevel(logrus.DebugLevel)
		}

		clientCertKeyBundle, err := cmd.Flags().GetString("cert-key-bundle")
		if err != nil {
			fmt.Printf("failed to get cert-key-bundle flag: %v\n", err)
			os.Exit(1)
		}
		clientCert, clientKey, err := util.GetUserCertAndKey(viper.GetString(defaults.ConfigClientCertKey), viper.GetString(defaults.ConfigClientKeyKey), clientCertKeyBundle, defaults.DefaultBundlePath)
		if err != nil {
			fmt.Printf("failed to get client cert and key: %v\n", err)
			os.Exit(1)
		}
		logger.Debugf("using user cert [%v] and key [%v]", clientCert, clientKey)

		client, err := client.NewClient(logger, quiet, clientCert, clientKey)
		if err != nil {
			fmt.Printf("failed to check conduit-server version: %v\n", err)
			os.Exit(1)
		}

		cvi, err := client.GetVersion()
		if err != nil {
			fmt.Printf("failed to check conduit-server version: %v\n", err)
			os.Exit(1)
		}
		cv := cvi.GetVersion()
		if cvi.GetModified() {
			cv += " (modified)"
		}
		fmt.Printf("conduit-server version: %s\n", cv)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
