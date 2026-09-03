package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
)

type FileSnapshot struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
	Tracked bool   `json:"tracked"`
}

type BaselineSnapshot struct {
	Available   bool                    `json:"available"`
	Dirty       bool                    `json:"dirty"`
	Head        string                  `json:"head,omitempty"`
	DiffDigest  string                  `json:"diffDigest,omitempty"`
	PreRunFiles map[string]FileSnapshot `json:"preRunFiles"`
	BaselineDir string                  `json:"baselineDir,omitempty"`
	CapturedAt  time.Time               `json:"capturedAt"`
}

// CaptureBaseline establishes a durable pre-run baseline of the working tree.
// It records file statuses, SHA256 hashes, and backs up pre-existing dirty files
// to backupDir so that any future mutation can be safely distinguished from
// pre-existing user changes and safely restored without blanket destructive resets.
func CaptureBaseline(ctx context.Context, root string, backupDir string) (BaselineSnapshot, error) {
	snapshot := BaselineSnapshot{
		PreRunFiles: map[string]FileSnapshot{},
		BaselineDir: backupDir,
		CapturedAt:  time.Now().UTC(),
	}

	head, err := run(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return snapshot, nil
	}
	snapshot.Available = true
	snapshot.Head = strings.TrimSpace(head)

	statusOut, err := run(ctx, root, "status", "--porcelain=v1", "-uall", "--", ".", ":(exclude).keystone")
	if err != nil {
		return snapshot, fmt.Errorf("git baseline status: %w", err)
	}

	for _, line := range strings.Split(strings.Trim(statusOut, "\r\n"), "\n") {
		st, relPath, ok := parsePorcelainLine(line)
		if !ok || relPath == ".keystone" || strings.HasPrefix(relPath, ".keystone/") {
			continue
		}
		snapshot.Dirty = true

		fullPath := filepath.Join(root, relPath)
		info, statErr := os.Stat(fullPath)
		var size int64
		var sum string
		if statErr == nil && !info.IsDir() {
			size = info.Size()
			sum, _ = hashFile(fullPath)
			if backupDir != "" {
				_ = copyFile(fullPath, filepath.Join(backupDir, relPath), info.Mode())
			}
		}

		tracked := st != "??"
		snapshot.PreRunFiles[relPath] = FileSnapshot{
			Path:    relPath,
			Status:  st,
			SHA256:  sum,
			Size:    size,
			Tracked: tracked,
		}
	}

	diff, err := run(ctx, root, "diff", "HEAD", "--no-ext-diff", "--binary", "--", ".", ":(exclude).keystone")
	if err == nil {
		dSum := sha256.Sum256([]byte(diff))
		snapshot.DiffDigest = hex.EncodeToString(dSum[:])
	}

	return snapshot, nil
}

// DetectMutations compares the current repository working tree against the pre-run baseline.
// It distinguishes pre-existing uncommitted changes from mutations introduced during the current run.
func DetectMutations(ctx context.Context, root string, baseline BaselineSnapshot) ([]domain.FileMutation, error) {
	if !baseline.Available {
		return nil, nil
	}

	statusOut, err := run(ctx, root, "status", "--porcelain=v1", "-uall", "--", ".", ":(exclude).keystone")
	if err != nil {
		return nil, fmt.Errorf("git status for mutation detection: %w", err)
	}

	currentStatus := map[string]string{}
	for _, line := range strings.Split(strings.Trim(statusOut, "\r\n"), "\n") {
		st, relPath, ok := parsePorcelainLine(line)
		if !ok || relPath == ".keystone" || strings.HasPrefix(relPath, ".keystone/") {
			continue
		}
		currentStatus[relPath] = st
	}

	mutations := []domain.FileMutation{}

	// Check each currently modified/untracked file against the baseline
	for path, st := range currentStatus {
		fullPath := filepath.Join(root, path)
		info, statErr := os.Stat(fullPath)
		currentSum := ""
		if statErr == nil && !info.IsDir() {
			currentSum, _ = hashFile(fullPath)
		}

		pre, existedBefore := baseline.PreRunFiles[path]
		if !existedBefore {
			// File was NOT dirty before the run.
			// If it was tracked in HEAD, it was modified by the run.
			// If not in HEAD, it was created by the run.
			if statErr != nil {
				// File disappeared or was deleted
				mutations = append(mutations, domain.FileMutation{
					Path:        path,
					Action:      "deleted",
					PreExisting: false,
				})
				continue
			}

			// Check if file is tracked in git HEAD
			isTracked := false
			if _, catErr := run(ctx, root, "cat-file", "-e", "HEAD:"+path); catErr == nil {
				isTracked = true
			}

			if isTracked {
				mutations = append(mutations, domain.FileMutation{
					Path:          path,
					Action:        "modified",
					PreExisting:   false,
					CurrentSHA256: currentSum,
				})
			} else {
				mutations = append(mutations, domain.FileMutation{
					Path:          path,
					Action:        "created",
					PreExisting:   false,
					CurrentSHA256: currentSum,
				})
			}
			continue
		}

		// File WAS dirty or untracked before the run.
		// Check if its content has changed compared to baseline.
		if st == " D" || st == "D " || statErr != nil {
			mutations = append(mutations, domain.FileMutation{
				Path:         path,
				Action:       "deleted",
				PreExisting:  true,
				PreRunSHA256: pre.SHA256,
			})
			continue
		}

		if currentSum != pre.SHA256 {
			mutations = append(mutations, domain.FileMutation{
				Path:          path,
				Action:        "modified",
				PreExisting:   true,
				PreRunSHA256:  pre.SHA256,
				CurrentSHA256: currentSum,
			})
		}
	}

	// Check if any file that existed before was deleted
	for path, pre := range baseline.PreRunFiles {
		if _, stillThere := currentStatus[path]; stillThere {
			continue
		}
		fullPath := filepath.Join(root, path)
		if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
			mutations = append(mutations, domain.FileMutation{
				Path:         path,
				Action:       "deleted",
				PreExisting:  true,
				PreRunSHA256: pre.SHA256,
			})
		}
	}

	return mutations, nil
}

// RestoreMutations safely and non-destructively reverts forbidden mutations introduced
// during the run while strictly preserving pre-existing user work.
func RestoreMutations(ctx context.Context, root string, baseline BaselineSnapshot, mutations []domain.FileMutation) (int, error) {
	restoredCount := 0
	var firstErr error

	for i := range mutations {
		m := &mutations[i]
		fullPath := filepath.Join(root, m.Path)

		switch m.Action {
		case "created":
			// Safely remove file created by run
			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				m.Restored = false
				m.Error = fmt.Sprintf("failed to remove created file: %v", err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			// Clean up empty parent directories up to root
			cleanEmptyDirs(root, filepath.Dir(fullPath))
			m.Restored = true
			restoredCount++

		case "modified":
			if !m.PreExisting {
				// Clean tracked file modified by run: restore from git HEAD
				if _, err := run(ctx, root, "checkout", "HEAD", "--", m.Path); err != nil {
					m.Restored = false
					m.Error = fmt.Sprintf("failed to checkout clean file: %v", err)
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				m.Restored = true
				restoredCount++
			} else {
				// Pre-existing dirty file modified by run: restore from pre-run backup
				if baseline.BaselineDir == "" {
					m.Restored = false
					m.Error = "no baseline backup directory available"
					continue
				}
				backupPath := filepath.Join(baseline.BaselineDir, m.Path)
				info, err := os.Stat(backupPath)
				if err != nil {
					m.Restored = false
					m.Error = fmt.Sprintf("pre-run backup not found: %v", err)
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				if err := copyFile(backupPath, fullPath, info.Mode()); err != nil {
					m.Restored = false
					m.Error = fmt.Sprintf("failed to restore pre-run backup: %v", err)
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				m.Restored = true
				restoredCount++
			}

		case "deleted":
			if !m.PreExisting {
				// Clean tracked file deleted by run: restore from git HEAD
				if _, err := run(ctx, root, "checkout", "HEAD", "--", m.Path); err != nil {
					m.Restored = false
					m.Error = fmt.Sprintf("failed to checkout deleted file: %v", err)
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				m.Restored = true
				restoredCount++
			} else {
				// Pre-existing dirty file deleted by run: restore from pre-run backup
				if baseline.BaselineDir == "" {
					m.Restored = false
					m.Error = "no baseline backup directory available"
					continue
				}
				backupPath := filepath.Join(baseline.BaselineDir, m.Path)
				info, err := os.Stat(backupPath)
				if err != nil {
					m.Restored = false
					m.Error = fmt.Sprintf("pre-run backup not found: %v", err)
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					m.Restored = false
					m.Error = err.Error()
					continue
				}
				if err := copyFile(backupPath, fullPath, info.Mode()); err != nil {
					m.Restored = false
					m.Error = fmt.Sprintf("failed to restore deleted pre-run file: %v", err)
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				m.Restored = true
				restoredCount++
			}
		}
	}

	return restoredCount, firstErr
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func cleanEmptyDirs(root, dir string) {
	cleanRoot := filepath.Clean(root)
	current := filepath.Clean(dir)
	for current != cleanRoot && strings.HasPrefix(current, cleanRoot) {
		entries, err := os.ReadDir(current)
		if err != nil || len(entries) > 0 {
			break
		}
		_ = os.Remove(current)
		current = filepath.Dir(current)
	}
}

func parsePorcelainLine(line string) (status string, path string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	if len(line) < 3 {
		return "", "", false
	}
	var st, rawPath string
	if len(line) >= 4 && line[2] == ' ' {
		st = line[:2]
		rawPath = strings.TrimSpace(line[3:])
	} else {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		st = parts[0]
		rawPath = strings.TrimSpace(parts[1])
	}
	if strings.Contains(rawPath, " -> ") {
		parts := strings.Split(rawPath, " -> ")
		rawPath = parts[len(parts)-1]
	}
	return st, filepath.ToSlash(filepath.Clean(rawPath)), true
}
