package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestPromptEditorNonInteractive(t *testing.T) {
	input := "hello world\n/sessions\n/exit\n"
	in := strings.NewReader(input)
	var out bytes.Buffer

	pe := NewPromptEditor(in, &out, "antigravity", "", "ready")

	line1, err := pe.ReadLine()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line1 != "hello world" {
		t.Fatalf("expected 'hello world', got %q", line1)
	}

	line2, err := pe.ReadLine()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line2 != "/sessions" {
		t.Fatalf("expected '/sessions', got %q", line2)
	}
}

func TestPrintBanner(t *testing.T) {
	var out bytes.Buffer
	PrintBanner(&out, "/Users/akshay/Desktop/code/losal", "antigravity", "heyakshaysaini@gmail.com", "Gemini 3.8 Flash (High)")

	s := out.String()
	for _, expected := range []string{
		"Keystone CLI 2.1.3",
		"heyakshaysaini@gmail.com",
		"antigravity",
		"Gemini 3.8 Flash",
	} {
		if !strings.Contains(s, expected) {
			t.Errorf("missing expected banner text %q in: %s", expected, s)
		}
	}
}
