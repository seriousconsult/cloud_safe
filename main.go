package main

import (
	"github.com/seriousconsult/cloud_safe/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
