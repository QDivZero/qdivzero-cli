// Package client builds the QDivZero API client from the local configuration.
package client

import (
	"context"
	"fmt"

	"github.com/QDivZero/qdivzero-cli/internal/config"
	"github.com/QDivZero/qdivzero-go"
)

// Factory returns a function that builds a configured qdivzero.API client.
// It fails with a helpful message when the user is not configured.
func Factory() func(ctx context.Context) (*qdivzero.API, error) {
	return func(ctx context.Context) (*qdivzero.API, error) {
		cfg, err := config.Read()
		if err != nil {
			return nil, err
		}
		if !config.IsConfigured(cfg) {
			return nil, fmt.Errorf("qdivzero is not configured; run 'qdivzero configure' first")
		}
		var opts []qdivzero.Option
		if cfg.PrivateBetaToken != "" {
			opts = append(opts, qdivzero.WithHeader("X-Private-Beta-Token", cfg.PrivateBetaToken))
		}
		return qdivzero.NewAPI(opts...)
	}
}
