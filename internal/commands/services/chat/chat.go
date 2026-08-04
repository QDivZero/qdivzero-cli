// Package chat implements the "qdivzero chat" service commands.
package chat

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/QDivZero/qdivzero-cli/internal/commands"
	"github.com/QDivZero/qdivzero-go"
)

// New returns the chat service command tree.
func New(deps *commands.Deps) *cobra.Command {
	var (
		model       string
		maxTokens   int
		temperature float32
		listModels  bool
		noStream    bool
	)
	cmd := &cobra.Command{
		Use:   "chat [prompt]",
		Short: "Chat with a model (interactive REPL when no prompt is given)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			api, err := deps.Client(ctx)
			if err != nil {
				return err
			}
			if listModels {
				return printFixedModels(ctx, api, deps)
			}
			if model == "" {
				return fmt.Errorf("chat: --model is required (see 'qdivzero chat --list-models')")
			}
			if len(args) == 1 {
				return runOneShot(ctx, api, deps, model, args[0], maxTokens, temperature, !noStream)
			}
			return runREPL(ctx, api, deps, model, maxTokens, temperature, !noStream)
		},
	}
	cmd.Flags().StringVar(&model, "model", "", "model to chat with")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "max tokens to generate")
	cmd.Flags().Float32Var(&temperature, "temperature", 0, "sampling temperature")
	cmd.Flags().BoolVar(&listModels, "list-models", false, "list available models and exit")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "disable streaming")
	return cmd
}

// chat sends the request; when stream is true it prints deltas as they
// arrive, otherwise it prints the assembled response.
func chat(ctx context.Context, api *qdivzero.API, model, prompt string, maxTokens int, temperature float32, stream bool, deps *commands.Deps) error {
	body, err := buildRequest(model, prompt, stream, maxTokens, temperature)
	if err != nil {
		return err
	}
	resp, err := api.PostV1ChatCompletionsWithBody(ctx, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chat: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if stream {
		return parseSSE(resp.Body, func(delta string) {
			fmt.Fprint(deps.Stdout, delta)
		})
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("chat: read: %w", err)
	}
	fmt.Fprintln(deps.Stdout, string(raw))
	return nil
}

func runOneShot(ctx context.Context, api *qdivzero.API, deps *commands.Deps, model, prompt string, maxTokens int, temperature float32, stream bool) error {
	if err := chat(ctx, api, model, prompt, maxTokens, temperature, stream, deps); err != nil {
		return err
	}
	if stream {
		fmt.Fprintln(deps.Stdout)
	}
	return nil
}

func runREPL(ctx context.Context, api *qdivzero.API, deps *commands.Deps, model string, maxTokens int, temperature float32, stream bool) error {
	reader := bufio.NewReader(deps.Stdin)
	fmt.Fprintf(deps.Stdout, "Chatting with %s (empty line to exit)\n", model)
	for {
		fmt.Fprint(deps.Stdout, "> ")
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "" {
			return nil
		}
		if err := chat(ctx, api, model, strings.TrimSpace(line), maxTokens, temperature, stream, deps); err != nil {
			return err
		}
		fmt.Fprintln(deps.Stdout)
	}
}

func printFixedModels(ctx context.Context, api *qdivzero.API, deps *commands.Deps) error {
	resp, err := api.GetModelsFixedWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("models: %w", err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("models: status %d", resp.StatusCode())
	}
	var rows [][]string
	if resp.JSON200 != nil && resp.JSON200.Models != nil {
		for i := range *resp.JSON200.Models {
			m := &(*resp.JSON200.Models)[i]
			rows = append(rows, []string{str(m.RepoId), str(m.PipelineTag)})
		}
	}
	return deps.Render.Render([]string{"MODEL", "PIPELINE"}, rows, resp.JSON200)
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
