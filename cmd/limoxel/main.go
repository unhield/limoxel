package main

import (
	"fmt"
	"os"

	capcli "github.com/unhield/limoxel/internal/capabilities/cli"
	"github.com/unhield/limoxel/internal/cli"
	"github.com/unhield/limoxel/internal/version"
)

func main() {
	cfg, err := cli.NewConfig("limoxel", version.Version, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI config: %v\n", err)
		os.Exit(1)
	}

	boot, err := cli.NewBootstrap(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI bootstrap: %v\n", err)
		os.Exit(1)
	}

	_, err = boot.Initialize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing CLI bootstrap: %v\n", err)
		os.Exit(1)
	}

	app := capcli.NewApp()
	exitCode := app.Run(os.Args[1:])
	os.Exit(exitCode)
}
