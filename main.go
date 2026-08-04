package main

import (
	"os"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
)

var version = "dev"

func main() {
	if err := commands.Execute(version); err != nil {
		os.Exit(1)
	}
}
