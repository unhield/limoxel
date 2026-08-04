package main

import (
	"fmt"
	"os"

	"github.com/unhield/limoxel/internal/cli"
)

func main() {
	cfg, err := cli.NewConfig("limoxel", "1.0.0", ".")
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
}
