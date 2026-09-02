package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
	"github.com/anonyxhappie/keystone/v2/internal/state"
)

func SaveText(s state.Store, kind, text string) (domain.Artifact, error) {
	sum := sha256.Sum256([]byte(text))
	id := "ART-" + hex.EncodeToString(sum[:8])
	a := domain.Artifact{ID: id, Kind: kind, Digest: hex.EncodeToString(sum[:]), Path: "artifacts/" + id + ".txt", CreatedAt: time.Now().UTC()}
	if err := s.Write(a.Path, map[string]string{"content": redact(text)}); err != nil {
		return a, err
	}
	if err := s.Write("artifacts/"+id+".json", a); err != nil {
		return a, err
	}
	return a, nil
}
func redact(text string) string {
	for _, key := range []string{"api_key", "apikey", "token", "password", "secret"} {
		offset := 0
		for offset < len(text) {
			low := strings.ToLower(text)
			rel := strings.Index(low[offset:], key)
			if rel < 0 {
				break
			}
			i := offset + rel
			end := strings.IndexAny(text[i+len(key):], " \t,;\n")
			if end < 0 {
				end = len(text) - i - len(key)
			}
			end += i + len(key)
			text = text[:i] + key + "=[REDACTED]" + text[end:]
			offset = i + len(key) + len("=[REDACTED]")
		}
	}
	return text
}
func Read(s state.Store, a domain.Artifact) (string, error) {
	var content struct {
		Content string `json:"content"`
	}
	if err := s.Read(a.Path, &content); err != nil {
		return "", fmt.Errorf("read artifact %s: %w", a.ID, err)
	}
	return content.Content, nil
}
