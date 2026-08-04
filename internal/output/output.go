// Package output renders command results as aligned tables or JSON.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// Renderer renders command results. The JSON provider is called per render,
// so the --json flag can toggle after construction.
type Renderer struct {
	jsonFn func() bool
	writer io.Writer
}

// New creates a renderer. jsonFn is called on every Render to decide between
// JSON and table output.
func New(jsonFn func() bool, w io.Writer) *Renderer {
	return &Renderer{jsonFn: jsonFn, writer: w}
}

// Render writes the result: the JSON payload when in JSON mode, otherwise a
// table built from headers and rows.
func (r *Renderer) Render(headers []string, rows [][]string, jsonValue any) error {
	if r.jsonFn() {
		data, err := json.MarshalIndent(jsonValue, "", "  ")
		if err != nil {
			return fmt.Errorf("output: marshal: %w", err)
		}
		_, err = fmt.Fprintln(r.writer, string(data))
		return err
	}
	return r.Table(headers, rows)
}

// Table writes an aligned table. Empty cells render as "-".
func (r *Renderer) Table(headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(r.writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, joinTab(headers)); err != nil {
		return err
	}
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range headers {
			if i < len(row) && row[i] != "" {
				cells[i] = row[i]
			} else {
				cells[i] = "-"
			}
		}
		if _, err := fmt.Fprintln(tw, joinTab(cells)); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func joinTab(cells []string) string {
	out := ""
	for i, c := range cells {
		if i > 0 {
			out += "\t"
		}
		out += c
	}
	return out
}
