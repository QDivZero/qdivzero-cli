package commands

import (
	"context"
	"io"

	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-cli/internal/output"
	"github.com/QDivZero/qdivzero-go"
)

// Deps carries the shared dependencies injected into every command. It is
// immutable after construction; service packages never import each other.
type Deps struct {
	Version string
	Client  func(ctx context.Context) (*qdivzero.API, error)
	Render  *output.Renderer
	Config  *config.Credentials
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}
