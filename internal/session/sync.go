package session

import (
	"io"
	"os"
	"path/filepath"
)

// EnsureAntigravitySessionAvailable checks if an Antigravity conversation is present
// in antigravity-cli. If it exists only in the IDE path (~/.gemini/antigravity),
// it copies the database so that agy CLI can resume it seamlessly.
func EnsureAntigravitySessionAvailable(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	cliDb := filepath.Join(home, ".gemini", "antigravity-cli", "conversations", sessionID+".db")
	if _, err := os.Stat(cliDb); err == nil {
		return true
	}

	ideDb := filepath.Join(home, ".gemini", "antigravity", "conversations", sessionID+".db")
	if _, err := os.Stat(ideDb); err != nil {
		return false
	}

	// Copy from IDE to CLI
	cliDir := filepath.Join(home, ".gemini", "antigravity-cli", "conversations")
	_ = os.MkdirAll(cliDir, 0755)

	src, err := os.Open(ideDb)
	if err != nil {
		return false
	}
	defer src.Close()

	dst, err := os.OpenFile(cliDb, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return false
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(cliDb)
		return false
	}

	// Also copy annotations pbtxt if present
	ideAnn := filepath.Join(home, ".gemini", "antigravity", "annotations", sessionID+".pbtxt")
	if _, err := os.Stat(ideAnn); err == nil {
		cliAnnDir := filepath.Join(home, ".gemini", "antigravity-cli", "annotations")
		_ = os.MkdirAll(cliAnnDir, 0755)
		cliAnn := filepath.Join(cliAnnDir, sessionID+".pbtxt")
		if srcAnn, err := os.Open(ideAnn); err == nil {
			defer srcAnn.Close()
			if dstAnn, err := os.OpenFile(cliAnn, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
				defer dstAnn.Close()
				_, _ = io.Copy(dstAnn, srcAnn)
			}
		}
	}

	return true
}
