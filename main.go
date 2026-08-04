package main

import (
	"os"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-cli/internal/commands/services/accounts"
	"github.com/QDivZero/qdivzero-cli/internal/commands/services/apikeys"
	"github.com/QDivZero/qdivzero-cli/internal/commands/services/chat"
	"github.com/QDivZero/qdivzero-cli/internal/commands/services/instances"
	"github.com/QDivZero/qdivzero-cli/internal/commands/services/models"
	"github.com/QDivZero/qdivzero-cli/internal/commands/services/serving"
)

var version = "dev"

func main() {
	if err := commands.Execute(version, instances.New, serving.New, models.New, apikeys.New, accounts.New, chat.New); err != nil {
		os.Exit(1)
	}
}
