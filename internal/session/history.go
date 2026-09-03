package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/ui"
)

// ConversationMessage represents a parsed turn in the conversation.
type ConversationMessage struct {
	Role      string    `json:"role"` // "user", "assistant", "tool", "system"
	Content   string    `json:"content"`
	ToolCalls []string  `json:"tool_calls,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// LoadConversationHistory loads recent messages for a given harness session.
func LoadConversationHistory(harnessName, sessionID string) []ConversationMessage {
	if sessionID == "" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// Try Antigravity transcript paths
	paths := []string{
		filepath.Join(home, ".gemini", "antigravity", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl"),
		filepath.Join(home, ".gemini", "antigravity-cli", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl"),
		filepath.Join(home, ".gemini", "antigravity", "brain", sessionID, ".system_generated", "logs", "transcript_full.jsonl"),
	}

	for _, p := range paths {
		if msgs, err := parseTranscriptJSONL(p); err == nil && len(msgs) > 0 {
			return msgs
		}
	}

	return nil
}

func parseTranscriptJSONL(path string) ([]ConversationMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var messages []ConversationMessage
	scanner := bufio.NewScanner(f)
	// Buffer up to 1MB per line
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var step struct {
			Type      string `json:"type"`
			Source    string `json:"source"`
			Content   string `json:"content"`
			Thinking  string `json:"thinking"`
			CreatedAt string `json:"created_at"`
			ToolCalls []struct {
				Name string         `json:"name"`
				Args map[string]any `json:"args"`
			} `json:"tool_calls"`
		}
		if err := json.Unmarshal(line, &step); err != nil {
			continue
		}

		t, _ := time.Parse(time.RFC3339, step.CreatedAt)

		switch step.Type {
		case "USER_INPUT":
			clean := cleanUserRequest(step.Content)
			if clean != "" {
				messages = append(messages, ConversationMessage{
					Role:      "user",
					Content:   clean,
					Timestamp: t,
				})
			}
		case "PLANNER_RESPONSE":
			var toolSummaries []string
			for _, tc := range step.ToolCalls {
				summary := tc.Name
				if len(tc.Args) > 0 {
					argPairs := []string{}
					for k, v := range tc.Args {
						vStr := fmt.Sprintf("%v", v)
						vStr = strings.Trim(vStr, "\"")
						if len(vStr) > 40 {
							vStr = vStr[:37] + "..."
						}
						argPairs = append(argPairs, fmt.Sprintf("%s=%s", k, vStr))
					}
					summary += "(" + strings.Join(argPairs, ", ") + ")"
				}
				toolSummaries = append(toolSummaries, summary)
			}
			content := strings.TrimSpace(step.Content)
			if content != "" || len(toolSummaries) > 0 {
				messages = append(messages, ConversationMessage{
					Role:      "assistant",
					Content:   content,
					ToolCalls: toolSummaries,
					Timestamp: t,
				})
			}
		}
	}

	return messages, nil
}

func cleanUserRequest(raw string) string {
	s := raw
	if idx := strings.Index(s, "<USER_REQUEST>"); idx != -1 {
		s = s[idx+len("<USER_REQUEST>"):]
	}
	if idx := strings.Index(s, "</USER_REQUEST>"); idx != -1 {
		s = s[:idx]
	}
	// Strip checkpoint summaries
	if idx := strings.Index(s, "<CONTEXT_SUMMARY>"); idx != -1 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

// RenderConversationHistory prints formatted conversation history to writer.
func RenderConversationHistory(out io.Writer, harnessName, sessionID string, maxTurns int) {
	messages := LoadConversationHistory(harnessName, sessionID)
	if len(messages) == 0 {
		return
	}

	fmt.Fprintf(out, "\n%s── Conversation History: %s (%s) ──%s\n", ui.Bold+ui.Cyan, sessionID, harnessName, ui.Reset)

	start := 0
	if maxTurns > 0 && len(messages) > maxTurns {
		start = len(messages) - maxTurns
		fmt.Fprintf(out, "%s  (... %d earlier turns omitted ...) %s\n", ui.Dim, start, ui.Reset)
	}

	for _, msg := range messages[start:] {
		switch msg.Role {
		case "user":
			lines := strings.Split(msg.Content, "\n")
			preview := lines[0]
			if len(preview) > 90 {
				preview = preview[:87] + "..."
			}
			fmt.Fprintf(out, "\n%s👤 User:%s %s\n", ui.Bold+ui.Yellow, ui.Reset, preview)
			if len(lines) > 1 {
				for _, l := range lines[1:] {
					if strings.TrimSpace(l) == "" {
						continue
					}
					if len(l) > 90 {
						l = l[:87] + "..."
					}
					fmt.Fprintf(out, "   %s%s%s\n", ui.Dim, l, ui.Reset)
				}
			}
		case "assistant":
			if msg.Content != "" {
				lines := strings.Split(msg.Content, "\n")
				preview := lines[0]
				if len(preview) > 90 {
					preview = preview[:87] + "..."
				}
				fmt.Fprintf(out, "%s🤖 %s:%s %s\n", ui.Bold+ui.Green, strings.Title(harnessName), ui.Reset, preview)
			}
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(out, "   %s↳ ⚡ %s%s\n", ui.Dim+ui.Cyan, tc, ui.Reset)
			}
		}
	}
	fmt.Fprintf(out, "%s─────────────────────────────────────────────────────────────────%s\n\n", ui.Bold+ui.Cyan, ui.Reset)
}
