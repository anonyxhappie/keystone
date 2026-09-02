package observation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJournalSequencesIdempotencyAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	key := "op-1"
	if err := j.Append(Event{RunID: "run", Type: "one", IdempotencyKey: key}); err != nil {
		t.Fatal(err)
	}
	if err := j.Append(Event{RunID: "run", Type: "one", IdempotencyKey: key}); err != nil {
		t.Fatal(err)
	}
	events, err := j.Events()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Sequence != 1 || events[0].Type != "one" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestJournalRejectsConflictingIdempotencyReuse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	j, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Append(Event{RunID: "run", Type: "one", IdempotencyKey: "op-1"}); err != nil {
		t.Fatal(err)
	}
	if err := j.Append(Event{RunID: "run", Type: "different", IdempotencyKey: "op-1"}); err == nil {
		t.Fatal("expected conflicting idempotency reuse to fail")
	}
}

func TestSeparateJournalInstancesShareSequenceSafely(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Append(Event{RunID: "run", Type: "one"}); err != nil {
		t.Fatal(err)
	}
	if err := second.Append(Event{RunID: "run", Type: "two"}); err != nil {
		t.Fatal(err)
	}
	events, err := first.Events()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Sequence != 1 || events[1].Sequence != 2 {
		t.Fatalf("separate journal instances reused sequence: %+v", events)
	}
}

func TestJournalRejectsMalformedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, []byte("not-json\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("expected malformed journal error")
	}
}
