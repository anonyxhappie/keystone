package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// CommandItem defines a suggestion item for the dropdown.
type CommandItem struct {
	Command     string
	Description string
	InsertText  string
	Immediate   bool
}

// DefaultSlashCommands lists all built-in slash commands.
var DefaultSlashCommands = []CommandItem{
	{Command: "/sessions", Description: "List conversations across Keystone & harnesses", InsertText: "/sessions", Immediate: true},
	{Command: "/resume", Description: "Resume a conversation by index or session ID", InsertText: "/resume ", Immediate: false},
	{Command: "/new", Description: "Start a fresh conversation (reset active session)", InsertText: "/new", Immediate: true},
	{Command: "/harness", Description: "Switch active harness (antigravity, codex, auto)", InsertText: "/harness ", Immediate: false},
	{Command: "/projects", Description: "List local code projects and workspaces", InsertText: "/projects", Immediate: true},
	{Command: "/project", Description: "Switch active workspace directory", InsertText: "/project ", Immediate: false},
	{Command: "/status", Description: "Inspect project state, lifecycle, and checkpoints", InsertText: "/status", Immediate: true},
	{Command: "/verify", Description: "Run deterministic validation checks on demand", InsertText: "/verify", Immediate: true},
	{Command: "/review", Description: "Inspect supervisor findings, drift, and advice", InsertText: "/review", Immediate: true},
	{Command: "/replay", Description: "Replay execution events of a previous run", InsertText: "/replay ", Immediate: false},
	{Command: "/doctor", Description: "Check environment, harness health, and tools", InsertText: "/doctor", Immediate: true},
	{Command: "/clear", Description: "Clear terminal screen", InsertText: "/clear", Immediate: true},
	{Command: "/exit", Description: "Exit the interactive session", InsertText: "/exit", Immediate: true},
}

// SuggestionProvider returns dynamic suggestions given the current prompt line.
type SuggestionProvider func(input string) []CommandItem

// PromptEditor provides rich interactive line editing with scrolling autocomplete dropdown.
type PromptEditor struct {
	in                 io.Reader
	out                io.Writer
	fallbackScanner    *bufio.Scanner
	suggestionProvider SuggestionProvider
	harnessName        string
	activeSession      string
	statusText         string
	history            []string
	historyIndex       int
}

// NewPromptEditor creates a new interactive prompt editor.
func NewPromptEditor(in io.Reader, out io.Writer, harnessName, activeSession, statusText string) *PromptEditor {
	return &PromptEditor{
		in:              in,
		out:             out,
		fallbackScanner: bufio.NewScanner(in),
		harnessName:     harnessName,
		activeSession:   activeSession,
		statusText:      statusText,
		suggestionProvider: func(input string) []CommandItem {
			if !strings.HasPrefix(input, "/") {
				return nil
			}
			lower := strings.ToLower(input)
			var matched []CommandItem
			for _, c := range DefaultSlashCommands {
				if strings.HasPrefix(strings.ToLower(c.Command), lower) {
					matched = append(matched, c)
				}
			}
			return matched
		},
	}
}

// SetSuggestionProvider sets a custom suggestion function.
func (pe *PromptEditor) SetSuggestionProvider(provider SuggestionProvider) {
	pe.suggestionProvider = provider
}

// SetContext updates active harness and session info for prompt display.
func (pe *PromptEditor) SetContext(harnessName, sessionID, statusText string) {
	pe.harnessName = harnessName
	pe.activeSession = sessionID
	pe.statusText = statusText
}

// ReadLine reads a line interactively with real-time dropdown and scroll support.
func (pe *PromptEditor) ReadLine() (string, error) {
	file, isFile := pe.in.(*os.File)
	if !isFile || !term.IsTerminal(int(file.Fd())) {
		if !pe.fallbackScanner.Scan() {
			if err := pe.fallbackScanner.Err(); err != nil {
				return "", err
			}
			return "", io.EOF
		}
		return strings.TrimSpace(pe.fallbackScanner.Text()), nil
	}

	fd := int(file.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		if !pe.fallbackScanner.Scan() {
			return "", io.EOF
		}
		return strings.TrimSpace(pe.fallbackScanner.Text()), nil
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	var buffer []rune
	cursorPos := 0
	selectedIndex := 0
	scrollOffset := 0
	menuDismissed := false
	renderedExtraLines := 0

	clearMenuLines := func() {
		if renderedExtraLines > 0 {
			// Clear all rendered extra lines from cursor down
			fmt.Fprint(pe.out, "\r\n\033[J")
			// Move cursor back up to prompt line
			fmt.Fprintf(pe.out, "\033[%dA\r", 1)
			renderedExtraLines = 0
		}
	}

	getSuggestions := func() []CommandItem {
		if menuDismissed {
			return nil
		}
		input := string(buffer)
		if pe.suggestionProvider != nil {
			return pe.suggestionProvider(input)
		}
		return nil
	}

	const maxVisible = 7

	redraw := func() {
		// Clean up dropdown from previous draw
		clearMenuLines()

		// Get terminal width to prevent line wrapping
		termWidth := 80
		if w, _, err := term.GetSize(fd); err == nil && w > 30 {
			termWidth = w
		}

		// Re-render prompt line
		fmt.Fprint(pe.out, "\r\033[2K")
		fmt.Fprintf(pe.out, "%s> %s%s", Bold+Cyan, Reset, string(buffer))

		matches := getSuggestions()
		if len(matches) > 0 {
			// Constrain selectedIndex
			if selectedIndex < 0 {
				selectedIndex = len(matches) - 1
			} else if selectedIndex >= len(matches) {
				selectedIndex = 0
			}

			// Adjust scroll offset to keep selectedIndex in viewport
			if selectedIndex < scrollOffset {
				scrollOffset = selectedIndex
			} else if selectedIndex >= scrollOffset+maxVisible {
				scrollOffset = selectedIndex - maxVisible + 1
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}
			if scrollOffset > len(matches)-maxVisible && len(matches) > maxVisible {
				scrollOffset = len(matches) - maxVisible
			}

			// Top indicator if scrolled down
			fmt.Fprint(pe.out, "\r\n\033[2K")
			renderedExtraLines++
			if scrollOffset > 0 {
				fmt.Fprintf(pe.out, "\r\n\033[2K  %s↑ %d more%s", Dim, scrollOffset, Reset)
				renderedExtraLines++
			}

			// Render visible window of items
			endIndex := scrollOffset + maxVisible
			if endIndex > len(matches) {
				endIndex = len(matches)
			}

			for i := scrollOffset; i < endIndex; i++ {
				item := matches[i]
				prefix := "  "
				cmdStyle := Cyan
				descStyle := Dim
				if i == selectedIndex {
					prefix = Bold + Cyan + "> " + Reset
					cmdStyle = Bold + White
					descStyle = White
				}

				cmdCol := fmt.Sprintf("%-28s", item.Command)
				descCol := item.Description
				availDesc := termWidth - 32
				if availDesc > 10 && len(descCol) > availDesc {
					descCol = descCol[:availDesc-3] + "..."
				}

				fmt.Fprintf(pe.out, "\r\n\033[2K%s%s %s%s%s", prefix, cmdStyle+cmdCol+Reset, descStyle, descCol, Reset)
				renderedExtraLines++
			}

			// Bottom indicator if more items below
			if endIndex < len(matches) {
				fmt.Fprintf(pe.out, "\r\n\033[2K  %s↓ %d more%s", Dim, len(matches)-endIndex, Reset)
				renderedExtraLines++
			}

			// Footer status and shortcuts
			status := pe.harnessName
			if pe.statusText != "" {
				status += " · " + pe.statusText
			}
			fmt.Fprintf(pe.out, "\r\n\033[2K\r\n\033[2K%s↑/↓ Navigate · enter Select · tab Complete%s", Dim, Reset)
			pad := termWidth - 45 - len(status)
			if pad < 2 {
				pad = 2
			}
			fmt.Fprintf(pe.out, "\r\n\033[2K%sesc to cancel%s%s%s%s%s", Dim, Reset, strings.Repeat(" ", pad), Dim, status, Reset)
			renderedExtraLines += 3

			// Return cursor back up to prompt line at cursorPos
			fmt.Fprintf(pe.out, "\033[%dA\r\033[%dC", renderedExtraLines, 2+cursorPos)
		} else {
			// Position cursor on prompt line
			fmt.Fprintf(pe.out, "\r\033[%dC", 2+cursorPos)
		}
	}

	redraw()

	for {
		ev, err := readKeyEvent(fd, pe.in)
		if err != nil {
			clearMenuLines()
			fmt.Fprintln(pe.out)
			return "", err
		}

		switch ev.Type {
		case KeyCtrlC:
			clearMenuLines()
			fmt.Fprint(pe.out, "\r\n")
			return "/exit", nil

		case KeyCtrlD:
			if len(buffer) == 0 {
				clearMenuLines()
				fmt.Fprint(pe.out, "\r\n")
				return "/exit", nil
			}

		case KeyEnter:
			matches := getSuggestions()
			if len(matches) > 0 && selectedIndex >= 0 && selectedIndex < len(matches) {
				selected := matches[selectedIndex]
				if selected.Immediate {
					clearMenuLines()
					result := selected.InsertText
					if result == "" {
						result = selected.Command
					}
					fmt.Fprint(pe.out, "\r\033[2K")
					fmt.Fprintf(pe.out, "%s> %s%s\r\n", Bold+Cyan, Reset, result)
					pe.history = append(pe.history, result)
					return result, nil
				}
				// Non-immediate: insert command onto prompt line for argument entry
				ins := selected.InsertText
				if ins == "" {
					ins = selected.Command + " "
				}
				buffer = []rune(ins)
				cursorPos = len(buffer)
				selectedIndex = 0
				scrollOffset = 0
				menuDismissed = false
				redraw()
				continue
			}

			clearMenuLines()
			result := string(buffer)
			fmt.Fprint(pe.out, "\r\033[2K")
			fmt.Fprintf(pe.out, "%s> %s%s\r\n", Bold+Cyan, Reset, result)
			if len(strings.TrimSpace(result)) > 0 {
				pe.history = append(pe.history, result)
			}
			return result, nil

		case KeyTab:
			matches := getSuggestions()
			if len(matches) > 0 && selectedIndex >= 0 && selectedIndex < len(matches) {
				selected := matches[selectedIndex]
				ins := selected.InsertText
				if ins == "" {
					ins = selected.Command + " "
				}
				buffer = []rune(ins)
				cursorPos = len(buffer)
				selectedIndex = 0
				scrollOffset = 0
				menuDismissed = false
				redraw()
				continue
			}

		case KeyEscape:
			menuDismissed = true
			if string(buffer) == "/" {
				buffer = nil
				cursorPos = 0
			}
			redraw()
			continue

		case KeyUp:
			matches := getSuggestions()
			if len(matches) > 0 {
				selectedIndex--
				if selectedIndex < 0 {
					selectedIndex = len(matches) - 1
				}
				redraw()
				continue
			}
			// History navigation
			if len(pe.history) > 0 {
				if pe.historyIndex < len(pe.history) {
					pe.historyIndex++
				}
				item := pe.history[len(pe.history)-pe.historyIndex]
				buffer = []rune(item)
				cursorPos = len(buffer)
				redraw()
			}
			continue

		case KeyDown:
			matches := getSuggestions()
			if len(matches) > 0 {
				selectedIndex++
				if selectedIndex >= len(matches) {
					selectedIndex = 0
				}
				redraw()
				continue
			}
			// History navigation
			if pe.historyIndex > 1 {
				pe.historyIndex--
				item := pe.history[len(pe.history)-pe.historyIndex]
				buffer = []rune(item)
				cursorPos = len(buffer)
				redraw()
			} else if pe.historyIndex == 1 {
				pe.historyIndex = 0
				buffer = nil
				cursorPos = 0
				redraw()
			}
			continue

		case KeyLeft:
			if cursorPos > 0 {
				cursorPos--
				redraw()
			}
			continue

		case KeyRight:
			if cursorPos < len(buffer) {
				cursorPos++
				redraw()
			}
			continue

		case KeyHome:
			cursorPos = 0
			redraw()
			continue

		case KeyEnd:
			cursorPos = len(buffer)
			redraw()
			continue

		case KeyBackspace:
			if cursorPos > 0 {
				buffer = append(buffer[:cursorPos-1], buffer[cursorPos:]...)
				cursorPos--
				selectedIndex = 0
				scrollOffset = 0
				menuDismissed = false
				redraw()
			}
			continue

		case KeyDelete:
			if cursorPos < len(buffer) {
				buffer = append(buffer[:cursorPos], buffer[cursorPos+1:]...)
				selectedIndex = 0
				scrollOffset = 0
				menuDismissed = false
				redraw()
			}
			continue

		case KeyRune:
			r := ev.Rune
			buffer = append(buffer[:cursorPos], append([]rune{r}, buffer[cursorPos:]...)...)
			cursorPos++
			selectedIndex = 0
			scrollOffset = 0
			menuDismissed = false
			redraw()
			continue
		}
	}
}

// PrintBanner renders the visual welcome card matching the Antigravity/Claude Code aesthetics.
func PrintBanner(out io.Writer, root, harnessName, account, modelInfo string) {
	// 5-line colorful block-art logo representing the Keystone arch
	logo1 := "\033[38;5;214m  ▄██████▄  \033[0m"
	logo2 := "\033[38;5;220m █▀  ██  ▀█ \033[0m"
	logo3 := "\033[38;5;118m █ ▄████▄ █ \033[0m"
	logo4 := "\033[38;5;45m █▀      ▀█ \033[0m"
	logo5 := "\033[38;5;39m ▀████████▀ \033[0m"

	harnessLabel := harnessName
	if harnessLabel == "" {
		harnessLabel = "antigravity"
	}
	modelLabel := modelInfo
	if modelLabel == "" {
		modelLabel = "Gemini 3.8 Flash (Supervised Autonomy)"
	}
	accountLabel := account
	if accountLabel == "" {
		accountLabel = "Keystone Engineering Intelligence Layer"
	}

	displayRoot := root
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(root, home) {
		displayRoot = "~" + strings.TrimPrefix(root, home)
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s   %sKeystone CLI 2.1.3%s\n", logo1, Bold+White, Reset)
	fmt.Fprintf(out, "%s   %s%s%s\n", logo2, Dim, accountLabel, Reset)
	fmt.Fprintf(out, "%s   %s%s · %s%s\n", logo3, Cyan, harnessLabel, modelLabel, Reset)
	fmt.Fprintf(out, "%s   %s%s%s\n", logo4, Dim, displayRoot, Reset)
	fmt.Fprintf(out, "%s\n\n", logo5)
}
