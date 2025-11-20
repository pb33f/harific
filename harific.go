package main

import (
	"os"

	"github.com/pb33f/harific/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}