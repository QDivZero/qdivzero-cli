package chat

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// streamChunk is the OpenAI-compatible SSE chunk shape we parse.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// parseSSE reads event-stream lines and invokes onContent for each content
// delta. It stops at the [DONE] marker or EOF.
func parseSSE(r io.Reader, onContent func(string)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, c := range chunk.Choices {
			if c.Delta.Content != "" {
				onContent(c.Delta.Content)
			}
		}
	}
	return scanner.Err()
}

// buildRequest marshals an OpenAI-compatible chat request (the typed
// ChatRequestEnvelope in the SDK cannot carry text content — the spec types
// content as []int).
func buildRequest(model, prompt string, stream bool, maxTokens int, temperature float32) ([]byte, error) {
	req := map[string]any{
		"model":    model,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   stream,
	}
	if maxTokens > 0 {
		req["max_tokens"] = maxTokens
	}
	if temperature > 0 {
		req["temperature"] = temperature
	}
	return json.Marshal(req)
}
