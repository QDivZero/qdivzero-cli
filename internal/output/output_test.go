package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestTableRendersAlignedColumns(t *testing.T) {
	var buf bytes.Buffer
	r := New(func() bool { return false }, &buf)
	r.Table([]string{"ID", "NAME"}, [][]string{
		{"i-1", "gpu-a"},
		{"i-22", "gpu-b"},
	})
	got := buf.String()
	if !strings.Contains(got, "i-1") || !strings.Contains(got, "gpu-b") {
		t.Fatalf("table missing rows:\n%s", got)
	}
	if !strings.HasPrefix(got, "ID") {
		t.Fatalf("table missing header:\n%s", got)
	}
}

func TestTableEmptyValuesDash(t *testing.T) {
	var buf bytes.Buffer
	r := New(func() bool { return false }, &buf)
	r.Table([]string{"A", "B"}, [][]string{{"", "x"}})
	if !strings.Contains(buf.String(), "-") {
		t.Fatalf("empty cell should render as '-':\n%s", buf.String())
	}
}

func TestJSONModePrintsJSON(t *testing.T) {
	var buf bytes.Buffer
	r := New(func() bool { return true }, &buf)
	r.Render([]string{"ID"}, [][]string{{"i-1"}}, map[string]string{"id": "i-1"})
	var out map[string]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if out["id"] != "i-1" {
		t.Fatalf("unexpected JSON: %s", buf.String())
	}
}
