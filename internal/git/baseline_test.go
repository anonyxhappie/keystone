package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmds := [][]string{
		{"init"},
		{"config", "user.name", "Keystone Test"},
		{"config", "user.email", "test@keystone.local"},
		{"config", "commit.gpgsign", "false"},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
		}
	}
	_ = os.WriteFile(filepath.Join(root, "clean.txt"), []byte("clean content"), 0644)
	_ = os.WriteFile(filepath.Join(root, "pre_dirty.txt"), []byte("initial commit content"), 0644)
	commitCmds := [][]string{
		{"add", "clean.txt", "pre_dirty.txt"},
		{"commit", "-m", "initial commit"},
	}
	for _, args := range commitCmds {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
		}
	}
	return root
}

func TestCaptureBaselineCleanRepo(t *testing.T) {
	root := initTestRepo(t)
	backup := filepath.Join(root, ".keystone", "baselines", "test")
	ctx := context.Background()

	b, err := CaptureBaseline(ctx, root, backup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !b.Available || b.Dirty || len(b.PreRunFiles) != 0 {
		t.Fatalf("expected clean baseline: %+v", b)
	}
}

func TestCaptureBaselineDirtyRepo(t *testing.T) {
	root := initTestRepo(t)
	// Create dirty uncommitted changes: one modified tracked, one untracked
	_ = os.WriteFile(filepath.Join(root, "pre_dirty.txt"), []byte("user uncommitted dirty work"), 0644)
	_ = os.WriteFile(filepath.Join(root, "user_untracked.txt"), []byte("user new file"), 0644)

	backup := filepath.Join(root, ".keystone", "baselines", "test")
	ctx := context.Background()

	b, err := CaptureBaseline(ctx, root, backup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !b.Dirty || len(b.PreRunFiles) != 2 {
		t.Fatalf("expected dirty baseline with 2 files, got: %+v", b)
	}

	// Verify backups were created
	for _, path := range []string{"pre_dirty.txt", "user_untracked.txt"} {
		if _, err := os.Stat(filepath.Join(backup, path)); err != nil {
			t.Fatalf("backup file not created for %s: %v", path, err)
		}
	}
}

func TestDetectMutationsNoChangeOnDirtyRepo(t *testing.T) {
	root := initTestRepo(t)
	_ = os.WriteFile(filepath.Join(root, "pre_dirty.txt"), []byte("user uncommitted dirty work"), 0644)
	_ = os.WriteFile(filepath.Join(root, "user_untracked.txt"), []byte("user new file"), 0644)

	backup := filepath.Join(root, ".keystone", "baselines", "test")
	ctx := context.Background()

	b, err := CaptureBaseline(ctx, root, backup)
	if err != nil {
		t.Fatal(err)
	}

	// Harness runs without modifying anything
	mutations, err := DetectMutations(ctx, root, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(mutations) != 0 {
		t.Fatalf("expected 0 mutations on untouched dirty repo, got: %+v", mutations)
	}
}

func TestDetectMutationsAndRestorePreservingUserWork(t *testing.T) {
	root := initTestRepo(t)
	// User has pre-existing uncommitted work:
	userDirtyContent := []byte("user original uncommitted changes")
	userUntrackedContent := []byte("user original untracked file")
	_ = os.WriteFile(filepath.Join(root, "pre_dirty.txt"), userDirtyContent, 0644)
	_ = os.WriteFile(filepath.Join(root, "user_untracked.txt"), userUntrackedContent, 0644)

	backup := filepath.Join(root, ".keystone", "baselines", "test")
	ctx := context.Background()

	baseline, err := CaptureBaseline(ctx, root, backup)
	if err != nil {
		t.Fatal(err)
	}

	// Now simulate a rogue harness turn:
	// 1. Modifies clean tracked file clean.txt
	_ = os.WriteFile(filepath.Join(root, "clean.txt"), []byte("rogue modification"), 0644)
	// 2. Modifies user's already-dirty file pre_dirty.txt
	_ = os.WriteFile(filepath.Join(root, "pre_dirty.txt"), []byte("rogue overwrite of user dirty file"), 0644)
	// 3. Creates brand new forbidden file
	_ = os.MkdirAll(filepath.Join(root, "src", "generated"), 0755)
	_ = os.WriteFile(filepath.Join(root, "src", "generated", "forbidden.ts"), []byte("export const x = 1;"), 0644)

	mutations, err := DetectMutations(ctx, root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(mutations) != 3 {
		t.Fatalf("expected 3 mutations detected, got: %+v", mutations)
	}

	// Verify classifications
	hasCreated := false
	hasCleanMod := false
	hasPreDirtyMod := false
	for _, m := range mutations {
		if m.Path == "src/generated/forbidden.ts" && m.Action == "created" && !m.PreExisting {
			hasCreated = true
		}
		if m.Path == "clean.txt" && m.Action == "modified" && !m.PreExisting {
			hasCleanMod = true
		}
		if m.Path == "pre_dirty.txt" && m.Action == "modified" && m.PreExisting {
			hasPreDirtyMod = true
		}
	}
	if !hasCreated || !hasCleanMod || !hasPreDirtyMod {
		t.Fatalf("unexpected mutation classifications: %+v", mutations)
	}

	// Perform Safe Rollback
	restored, err := RestoreMutations(ctx, root, baseline, mutations)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if restored != 3 {
		t.Fatalf("expected 3 files restored, got %d", restored)
	}

	// Verify post-restoration state:
	// 1. clean.txt must be back to HEAD clean content
	cleanBytes, _ := os.ReadFile(filepath.Join(root, "clean.txt"))
	if string(cleanBytes) != "clean content" {
		t.Fatalf("clean.txt was not restored: %q", string(cleanBytes))
	}

	// 2. pre_dirty.txt must be back to userDirtyContent (NOT initial commit, but the user's pre-run dirty work!)
	dirtyBytes, _ := os.ReadFile(filepath.Join(root, "pre_dirty.txt"))
	if string(dirtyBytes) != string(userDirtyContent) {
		t.Fatalf("pre_dirty.txt did not preserve user dirty work: %q", string(dirtyBytes))
	}

	// 3. user_untracked.txt must be untouched
	untrackedBytes, _ := os.ReadFile(filepath.Join(root, "user_untracked.txt"))
	if string(untrackedBytes) != string(userUntrackedContent) {
		t.Fatalf("user_untracked.txt was touched: %q", string(untrackedBytes))
	}

	// 4. forbidden.ts and empty dirs must be deleted
	if _, err := os.Stat(filepath.Join(root, "src", "generated", "forbidden.ts")); !os.IsNotExist(err) {
		t.Fatalf("forbidden.ts was not deleted")
	}
	if _, err := os.Stat(filepath.Join(root, "src")); !os.IsNotExist(err) {
		t.Fatalf("empty directory src was not cleaned up")
	}

	// Confirm that after restoration, DetectMutations reports 0 mutations!
	postMutations, err := DetectMutations(ctx, root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(postMutations) != 0 {
		t.Fatalf("expected 0 mutations after restoration, got: %+v", postMutations)
	}
}

func TestDetectMutationsDeletionAndRestoration(t *testing.T) {
	root := initTestRepo(t)
	backup := filepath.Join(root, ".keystone", "baselines", "test-del")
	ctx := context.Background()

	baseline, err := CaptureBaseline(ctx, root, backup)
	if err != nil {
		t.Fatal(err)
	}

	// Harness deletes clean.txt
	_ = os.Remove(filepath.Join(root, "clean.txt"))

	mutations, err := DetectMutations(ctx, root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(mutations) != 1 || mutations[0].Action != "deleted" || mutations[0].Path != "clean.txt" {
		t.Fatalf("expected 1 deleted mutation for clean.txt, got: %+v", mutations)
	}

	restored, err := RestoreMutations(ctx, root, baseline, mutations)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if restored != 1 {
		t.Fatalf("expected 1 file restored, got %d", restored)
	}

	// Verify clean.txt is back
	bytes, err := os.ReadFile(filepath.Join(root, "clean.txt"))
	if err != nil || string(bytes) != "clean content" {
		t.Fatalf("clean.txt was not restored properly: %v, %q", err, string(bytes))
	}
}

func TestRestoreMutationsMissingBackupFailsSafely(t *testing.T) {
	root := initTestRepo(t)
	ctx := context.Background()

	// Baseline without backup dir
	baseline := BaselineSnapshot{
		Available:   true,
		BaselineDir: "", // missing backup
		PreRunFiles: map[string]FileSnapshot{
			"pre_dirty.txt": {Path: "pre_dirty.txt", SHA256: "oldhash", Tracked: true},
		},
	}

	mutations := []domain.FileMutation{
		{Path: "pre_dirty.txt", Action: "modified", PreExisting: true, CurrentSHA256: "newhash"},
	}

	restored, _ := RestoreMutations(ctx, root, baseline, mutations)
	if restored != 0 {
		t.Fatalf("expected 0 restored when backup is missing, got %d", restored)
	}
	if mutations[0].Restored {
		t.Fatalf("expected mutation to be marked not restored")
	}
	if mutations[0].Error == "" {
		t.Fatalf("expected mutation error explaining missing backup")
	}
}
