// Copyright 2026. Triad National Security, LLC. All rights reserved.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	clicmd "github.com/lanl/conduit/internal/cmd/cli"
)

func main() {

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		switch sig {
		case syscall.SIGINT:
			os.Exit(128 + 2)
		case syscall.SIGTERM:
			os.Exit(128 + 15)
		}
	}()

	clicmd.Execute()
}
