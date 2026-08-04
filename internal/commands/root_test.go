package commands

import (
	"errors"
	"strings"
	"testing"
)

func TestHintForError(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errors.New("instances list: status 401"), "qdivzero configure --token"},
		{errors.New("qdivzero is not configured; run 'qdivzero configure' first"), "qdivzero configure"},
		{errors.New("instances list: status 500"), ""},
		{errors.New("network error"), ""},
	}
	for _, c := range cases {
		got := hintForError(c.err)
		if c.want == "" && got != "" {
			t.Fatalf("hintForError(%q) = %q, want empty", c.err, got)
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Fatalf("hintForError(%q) = %q, want contains %q", c.err, got, c.want)
		}
	}
}
