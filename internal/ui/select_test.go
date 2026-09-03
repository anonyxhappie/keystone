package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestSelectNonInteractive(t *testing.T) {
	items := []SelectItem{
		{Title: "losal", Description: "/path/to/losal", Active: true},
		{Title: "keystone", Description: "/path/to/keystone", Active: false},
	}

	var out bytes.Buffer
	in := strings.NewReader("")

	idx, err := Select(in, &out, "Select a project:", items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Fatalf("expected -1 in non-interactive mode, got %d", idx)
	}

	s := out.String()
	if !strings.Contains(s, "Select a project:") || !strings.Contains(s, "losal") {
		t.Fatalf("missing expected items in output: %s", s)
	}
}
