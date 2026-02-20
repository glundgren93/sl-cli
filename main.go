package main

import (
	"os"

	"github.com/glundgren93/sl-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
