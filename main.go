package main

import (
	"os"

	"github.com/syself/caphcli/internal/cmd"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
