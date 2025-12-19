package main

import (
	"os"

	"github.com/hzerrad/cronic/internal/cmd"
)

func main() {
	cmd.SetOutput(os.Stdout, os.Stderr)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
