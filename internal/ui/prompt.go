package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// CommandItem defines a slash command with description for the autocomplete dropdown.
type CommandItem struct {
	Command     string
	Description string
}

// DefaultSlashCommands lists all built-in slash commands.
var DefaultSlashCommands = []CommandItem{
	{Command: "/sessions", Description: "List conversations across Keystone & harnesses"},
	{Command: "/resume", Description: "Resume a conversation by index or session ID"},
	{Command: "/new", Description: "Start a fresh conversation (reset active session)"},
	{Command: "/harness", Description: "Switch active harness (antigravity, codex, auto)"},
	{Command: "/projects", Description: "List local code projects and workspaces"},
	{Command: "/project", Description: "Switch active workspace directory"},
	{Command: "/status", Description: "Inspect project state, lifecycle, and checkpoints"},
	{Command: "/verify", Description: "Run deterministic validation checks on demand"},
	{Command: "/review", Description: "Inspect supervisor findings, drift, and advice"},
	{Command: "/replay", Description: "Replay execution events of a previous run"},
	{Command: "/doctor", Description: "Check environment, harness health, and tools"},
	{Command: "/clear", Description: "Clear terminal screen"},
	{Command: "/exit", Description: "Exit the interactive session"},
}

// PromptEditor provides rich interactive line editing with slash command autocomplete.
type PromptEditor struct {
	in              io.Reader
	out             io.Writer
	fallbackScanner *bufio.Scanner
	commands        []CommandItem
	harnessName     string
	activeSession   string
	statusText      string
	history         []string
	historyIndex    int
}

// NewPromptEditor creates a new interactive prompt editor.
func NewPromptEditor(in io.Reader, out io.Writer, harnessName, activeSession, statusText string) *PromptEditor {
	return &PromptEditor{
		in:              in,
		out:             out,
		fallbackScanner: bufio.NewScanner(in),
		commands:        DefaultSlashCommands,
		harnessName:     harnessName,
		activeSession:   activeSession,
		statusText:      statusText,
	}
}

// SetContext updates active harness and session info for prompt display.
func (pe *PromptEditor) SetContext(harnessName, sessionID, statusText string) {
	pe.harnessName = harnessName
	pe.activeSession = sessionID
	pe.statusText = statusText
}

// ReadLine reads a line interactively with real-time dropdown support.
func (pe *PromptEditor) ReadLine() (string, error) {
	file, isFile := pe.in.(*os.File)
	if !isFile || !term.IsTerminal(int(file.Fd())) {
		// Non-interactive fallback for pipes, scripts, and tests
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
		scanner := bufio.NewScanner(pe.in)
		if !scanner.Scan() {
			return "", io.EOF
		}
		return strings.TrimSpace(scanner.Text()), nil
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	var buffer []rune
	cursorPos := 0
	selectedIndex := 0
	renderedExtraLines := 0

	clearExtraLines := func() {
		if renderedExtraLines > 0 {
			for i := 0; i < renderedExtraLines; i++ {
				fmt.Fprint(pe.out, "\r\n\033[2K")
			}
			fmt.Fprintf(pe.out, "\033[%dA\r", renderedExtraLines)
			renderedExtraLines = 0
		}
	}

	filterCommands := func() []CommandItem {
		input := string(buffer)
		if !strings.HasPrefix(input, "/") {
			return nil
		}
		// If input contains arguments (space), do not show command dropdown
		if strings.Contains(input, " ") {
			return nil
		}
		var matched []CommandItem
		lower := strings.ToLower(input)
		for _, c := range pe.commands {
			if strings.HasPrefix(strings.ToLower(c.Command), lower) {
				matched = append(matched, c)
			}
		}
		return matched
	}

	redraw := func() {
		// Clear previously rendered dropdown lines below
		clearExtraLines()

		// Clear current prompt line and write prompt + buffer
		fmt.Fprint(pe.out, "\r\033[2K")
		fmt.Fprintf(pe.out, "%s> %s%s", Bold+Cyan, Reset, string(buffer))

		// Check if slash command dropdown should be shown
		matches := filterCommands()
		if len(matches) > 0 {
			if selectedIndex >= len(matches) {
				selectedIndex = 0
			}
			if selectedIndex < 0 {
				selectedIndex = len(matches) - 1
			}

			displayLimit := 5
			if len(matches) < displayLimit {
				displayLimit = len(matches)
			}

			// Render dropdown items
			fmt.Fprint(pe.out, "\r\n\033[2K")
			renderedExtraLines++

			for i := 0; i < displayLimit; i++ {
				item := matches[i]
				prefix := "  "
				cmdStyle := Cyan
				if i == selectedIndex {
					prefix = Bold + Cyan + "> " + Reset
					cmdStyle = Bold + White
				}
				fmt.Fprintf(pe.out, "\r\n\033[2K%s%-28s %s%s%s", prefix, cmdStyle+item.Command+Reset, Dim, item.Description, Reset)
				renderedExtraLines++
			}

			if len(matches) > displayLimit {
				fmt.Fprintf(pe.out, "\r\n\033[2K  %s↓ %d more%s", Dim, len(matches)-displayLimit, Reset)
				renderedExtraLines++
			}

			// Footer line
			status := pe.harnessName
			if pe.statusText != "" {
				status += " · " + pe.statusText
			}
			fmt.Fprintf(pe.out, "\r\n\033[2K\r\n\033[2K%s↑/↓ Navigate · enter Select · tab Complete%s", Dim, Reset)
			fmt.Fprintf(pe.out, "\r\n\033[2K%sesc to cancel%s                                                    %s%s%s", Dim, Reset, Dim, status, Reset)
			renderedExtraLines += 3

			// Move cursor back up to prompt line and place at cursorPos
			fmt.Fprintf(pe.out, "\033[%dA\r\033[%dC", renderedExtraLines, 2+cursorPos)
		} else {
			// Position cursor on prompt line
			fmt.Fprintf(pe.out, "\r\033[%dC", 2+cursorPos)
		}
	}

	redraw()

	buf := make([]byte, 16)
	for {
		n, err := pe.in.Read(buf)
		if err != nil {
			clearExtraLines()
			fmt.Fprintln(pe.out)
			return "", err
		}
		if n == 0 {
			continue
		}

		// Handle key combinations
		if n == 1 {
			b := buf[0]
			switch b {
			case 3: // Ctrl+C
				clearExtraLines()
				fmt.Fprint(pe.out, "\r\n")
				return "/exit", nil
			case 4: // Ctrl+D
				if len(buffer) == 0 {
					clearExtraLines()
					fmt.Fprint(pe.out, "\r\n")
					return "/exit", nil
				}
			case 13, 10: // Enter
				matches := filterCommands()
				if len(matches) > 0 && selectedIndex >= 0 && selectedIndex < len(matches) {
					// Complete selected command if user was browsing
					selected := matches[selectedIndex].Command
					buffer = []rune(selected)
					cursorPos = len(buffer)
				}
				clearExtraLines()
				fmt.Fprint(pe.out, "\r\n")
				result := string(buffer)
				if len(strings.TrimSpace(result)) > 0 {
					pe.history = append(pe.history, result)
				}
				return result, nil
			case 9: // Tab
				matches := filterCommands()
				if len(matches) > 0 && selectedIndex >= 0 && selectedIndex < len(matches) {
					selected := matches[selectedIndex].Command + " "
					buffer = []rune(selected)
					cursorPos = len(buffer)
					selectedIndex = 0
				}
				redraw()
				continue
			case 127, 8: // Backspace
				if cursorPos > 0 {
					buffer = append(buffer[:cursorPos-1], buffer[cursorPos:]...)
					cursorPos--
				}
				redraw()
				continue
			case 27: // Single Escape
				clearExtraLines()
				redraw()
				continue
			default:
				if b >= 32 && b <= 126 {
					r := rune(b)
					buffer = append(buffer[:cursorPos], append([]rune{r}, buffer[cursorPos:]...)...)
					cursorPos++
					redraw()
					continue
				}
			}
		}

		// Handle ANSI Escape Sequences (Arrows, PageUp/Down, etc.)
		if n >= 3 && buf[0] == 27 && buf[1] == 91 {
			matches := filterCommands()
			switch buf[2] {
			case 65: // Up Arrow
				if len(matches) > 0 {
					selectedIndex--
					if selectedIndex < 0 {
						selectedIndex = len(matches) - 1
					}
				} else if len(pe.history) > 0 {
					if pe.historyIndex < len(pe.history) {
						pe.historyIndex++
					}
					item := pe.history[len(pe.history)-pe.historyIndex]
					buffer = []rune(item)
					cursorPos = len(buffer)
				}
				redraw()
				continue
			case 66: // Down Arrow
				if len(matches) > 0 {
					selectedIndex++
					if selectedIndex >= len(matches) {
						selectedIndex = 0
					}
				} else if pe.historyIndex > 1 {
					pe.historyIndex--
					item := pe.history[len(pe.history)-pe.historyIndex]
					buffer = []rune(item)
					cursorPos = len(buffer)
				} else if pe.historyIndex == 1 {
					pe.historyIndex = 0
					buffer = nil
					cursorPos = 0
				}
				redraw()
				continue
			case 67: // Right Arrow
				if cursorPos < len(buffer) {
					cursorPos++
				}
				redraw()
				continue
			case 68: // Left Arrow
				if cursorPos > 0 {
					cursorPos--
				}
				redraw()
				continue
			}
		}

		// UTF-8 multi-byte characters
		r, size := utf8.DecodeRune(buf[:n])
		if r != utf8.RuneError && size > 0 && r >= 32 {
			buffer = append(buffer[:cursorPos], append([]rune{r}, buffer[cursorPos:]...)...)
			cursorPos++
			redraw()
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
