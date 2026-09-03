package session

import (
	"fmt"
	"io"
	"strings"

	"github.com/anonyxhappie/keystone/internal/ui"
)

// HarnessOption defines a selectable execution harness.
type HarnessOption struct {
	ID          string
	Title       string
	Description string
}

// ModelOption defines a selectable model for a given harness.
type ModelOption struct {
	ID          string
	Title       string
	Description string
}

var AvailableHarnesses = []HarnessOption{
	{
		ID:          "antigravity",
		Title:       "Antigravity",
		Description: "Google AI (Gemini 3.8 / 3.7 Flash, Claude)",
	},
	{
		ID:          "codex",
		Title:       "Codex",
		Description: "OpenAI (o3, o3-mini, GPT-4o)",
	},
	{
		ID:          "auto",
		Title:       "Auto",
		Description: "Supervised Autonomy (Auto-detect available harness)",
	},
}

var AntigravityModels = []ModelOption{
	{
		ID:          "gemini-3.8-flash-high",
		Title:       "Gemini 3.8 Flash (High)",
		Description: "Recommended · Google DeepMind",
	},
	{
		ID:          "gemini-3.7-flash-high",
		Title:       "Gemini 3.7 Flash (High)",
		Description: "Google DeepMind",
	},
	{
		ID:          "gemini-3.1-pro-high",
		Title:       "Gemini 3.1 Pro (High)",
		Description: "Google DeepMind",
	},
	{
		ID:          "claude-sonnet-4-6",
		Title:       "Claude Sonnet 4.6 (Thinking)",
		Description: "Anthropic via Antigravity",
	},
	{
		ID:          "claude-opus-4-6-thinking",
		Title:       "Claude Opus 4.6 (Thinking)",
		Description: "Anthropic via Antigravity",
	},
}

var CodexModels = []ModelOption{
	{
		ID:          "o3",
		Title:       "o3 (High)",
		Description: "Recommended · OpenAI Flagship Reasoning",
	},
	{
		ID:          "o3-mini",
		Title:       "o3-mini",
		Description: "OpenAI Fast Reasoning",
	},
	{
		ID:          "gpt-4.5",
		Title:       "gpt-4.5",
		Description: "OpenAI Research Model",
	},
	{
		ID:          "gpt-4o",
		Title:       "gpt-4o",
		Description: "OpenAI Multimodal",
	},
}

var AutoModels = []ModelOption{
	{
		ID:          "auto",
		Title:       "Default Provider Model",
		Description: "Selected automatically based on active harness",
	},
}

// PromptHarness presents an interactive selector for execution harness.
func PromptHarness(in io.Reader, out io.Writer, currentHarness string) string {
	items := make([]ui.SelectItem, len(AvailableHarnesses))
	for i, h := range AvailableHarnesses {
		items[i] = ui.SelectItem{
			Title:       h.Title,
			Description: h.Description,
			Active:      strings.EqualFold(h.ID, currentHarness),
		}
	}

	idx, _ := ui.Select(in, out, "Select execution harness:", items)
	if idx >= 0 && idx < len(AvailableHarnesses) {
		return AvailableHarnesses[idx].ID
	}
	if currentHarness != "" {
		return currentHarness
	}
	return "antigravity"
}

// PromptModel presents an interactive selector for models corresponding to the selected harness.
func PromptModel(in io.Reader, out io.Writer, harnessName, currentModel string) string {
	var models []ModelOption
	normHarness := strings.ToLower(harnessName)
	switch normHarness {
	case "codex":
		models = CodexModels
	case "auto":
		models = AutoModels
	default:
		models = AntigravityModels
	}

	items := make([]ui.SelectItem, len(models))
	for i, m := range models {
		items[i] = ui.SelectItem{
			Title:       m.Title,
			Description: m.Description,
			Active:      strings.EqualFold(m.ID, currentModel) || (currentModel == "" && i == 0),
		}
	}

	title := fmt.Sprintf("Select model for %s:", ModelDisplayName(normHarness, normHarness))
	if normHarness == "antigravity" {
		title = "Select model for Antigravity:"
	} else if normHarness == "codex" {
		title = "Select model for Codex:"
	} else if normHarness == "auto" {
		title = "Select model for Auto:"
	}

	idx, _ := ui.Select(in, out, title, items)
	if idx >= 0 && idx < len(models) {
		return models[idx].ID
	}
	if currentModel != "" {
		return currentModel
	}
	return models[0].ID
}
