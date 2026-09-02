package git

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

type State struct {
	Available    bool     `json:"available"`
	Dirty        bool     `json:"dirty"`
	ChangedFiles []string `json:"changedFiles,omitempty"`
	Head         string   `json:"head,omitempty"`
	DiffDigest   string   `json:"diffDigest,omitempty"`
	Error        string   `json:"error,omitempty"`
}

func Inspect(ctx context.Context, root string) State {
	state := State{}
	head, err := run(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		state.Error = err.Error()
		return state
	}
	state.Available = true
	state.Head = strings.TrimSpace(head)
	status, err := run(ctx, root, "status", "--porcelain", "--", ".", ":(exclude).keystone")
	if err != nil {
		state.Error = err.Error()
		return state
	}
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		state.Dirty = true
		if len(line) > 3 {
			name := strings.TrimSpace(line[3:])
			if name == ".keystone" || strings.HasPrefix(name, ".keystone/") {
				continue
			}
			state.ChangedFiles = append(state.ChangedFiles, name)
		}
	}
	diff, err := run(ctx, root, "diff", "HEAD", "--no-ext-diff", "--binary", "--", ".", ":(exclude).keystone")
	if err == nil {
		sum := sha256.Sum256([]byte(diff))
		state.DiffDigest = hex.EncodeToString(sum[:])
	}
	return state
}

func Diff(ctx context.Context, root string) (string, error) {
	return run(ctx, root, "diff", "--no-ext-diff", "--binary")
}

func run(ctx context.Context, root string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errout.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out.String(), nil
}
