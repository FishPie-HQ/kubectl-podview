package main

import (
	"os"

	"github.com/FishPie-HQ/kubectl-podview/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
