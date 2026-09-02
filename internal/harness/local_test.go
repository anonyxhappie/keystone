package harness

import (
	"context"
	"io"
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func TestLocalAdapterLifecycleAndRedaction(t *testing.T) {
	a := NewLocal(context.Background(), Config{Name: "fixture", Command: "sh", Args: []string{"-c", "read request; echo 'token=abc done'"}, TimeoutSeconds: 10})
	if err := a.Discover(); err != nil {
		t.Fatal(err)
	}
	id, err := a.Start(domain.WorkPacket{Objective: "test"})
	if err != nil || id == "" {
		t.Fatalf("start failed: %v %q", err, id)
	}
	items, err := a.Observe()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "COMPLETION_CLAIM" || items[0].Summary == "token=abc done" {
		t.Fatalf("unexpected observation: %+v", items)
	}
	if _, err := a.Observe(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
	status, err := a.Result()
	if err != nil || status != domain.StatusCompleted {
		t.Fatalf("unexpected result: %v %v", status, err)
	}
}

func TestLocalAdapterReportsProcessCrash(t *testing.T) {
	a := NewLocal(context.Background(), Config{Command: "sh", Args: []string{"-c", "read request; exit 7"}, TimeoutSeconds: 10})
	if _, err := a.Start(domain.WorkPacket{Objective: "crash"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Observe(); err != io.EOF {
		t.Fatalf("expected EOF after crash, got %v", err)
	}
	status, err := a.Result()
	if err != nil || status != domain.StatusFailed {
		t.Fatalf("unexpected crash result: %v %v", status, err)
	}
}
