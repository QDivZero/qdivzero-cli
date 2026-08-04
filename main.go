package main

import (
	"os"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-cli/internal/commands/services/instances"
)

var version = "dev"

func main() {
	if err := commands.Execute(version, instances.New); err != nil {
		os.Exit(1)
	}
}
