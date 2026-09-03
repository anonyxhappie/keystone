package ui

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// SelectItem represents an option in the interactive selection menu.
type SelectItem struct {
	Title       string
	Description string
	Active      bool
}

// Select displays an interactive terminal list for arrow-key navigation and Enter selection.
func Select(in io.Reader, out io.Writer, title string, items []SelectItem) (int, error) {
	if len(items) == 0 {
		return -1, nil
	}

	file, isFile := in.(*os.File)
	if !isFile || !term.IsTerminal(int(file.Fd())) {
		// Non-interactive fallback: print plain list
		fmt.Fprintf(out, "\n%s\n", Bold+title+Reset)
		for i, item := range items {
			mark := " "
			if item.Active {
				mark = "*"
			}
			fmt.Fprintf(out, " %s%2d. %-24s %s\n", mark, i+1, item.Title, item.Description)
		}
		fmt.Fprintln(out)
		return -1, nil
	}

	fd := int(file.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return -1, err
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	selectedIndex := 0
	scrollOffset := 0
	const maxVisible = 8
	renderedLines := 0

	termWidth := 80
	if w, _, err := term.GetSize(fd); err == nil && w > 30 {
		termWidth = w
	}

	clearMenu := func() {
		if renderedLines > 0 {
			fmt.Fprint(out, "\r\033[2K")
			for i := 0; i < renderedLines; i++ {
				fmt.Fprint(out, "\033[1A\r\033[2K")
			}
			renderedLines = 0
		}
	}

	render := func() {
		clearMenu()

		fmt.Fprintf(out, "\r\n%s%s%s", Bold+Cyan, title, Reset)
		renderedLines++

		if selectedIndex < scrollOffset {
			scrollOffset = selectedIndex
		} else if selectedIndex >= scrollOffset+maxVisible {
			scrollOffset = selectedIndex - maxVisible + 1
		}
		if scrollOffset < 0 {
			scrollOffset = 0
		}
		if scrollOffset > len(items)-maxVisible && len(items) > maxVisible {
			scrollOffset = len(items) - maxVisible
		}

		if scrollOffset > 0 {
			fmt.Fprintf(out, "\r\n  %s↑ %d more%s", Dim, scrollOffset, Reset)
			renderedLines++
		}

		endIndex := scrollOffset + maxVisible
		if endIndex > len(items) {
			endIndex = len(items)
		}

		for i := scrollOffset; i < endIndex; i++ {
			item := items[i]
			prefix := "    "
			cmdStyle := Cyan
			descStyle := Dim
			activeBadge := " "
			if item.Active {
				activeBadge = "*"
			}
			if i == selectedIndex {
				prefix = Bold + Cyan + "  > " + Reset
				cmdStyle = Bold + White
				descStyle = White
			}

			titleCol := fmt.Sprintf("%s%2d. %-24s", activeBadge, i+1, item.Title)
			descCol := item.Description
			availDesc := termWidth - 36
			if availDesc > 10 && len(descCol) > availDesc {
				descCol = descCol[:availDesc-3] + "..."
			}

			fmt.Fprintf(out, "\r\n%s%s %s%s%s", prefix, cmdStyle+titleCol+Reset, descStyle, descCol, Reset)
			renderedLines++
		}

		if endIndex < len(items) {
			fmt.Fprintf(out, "\r\n  %s↓ %d more%s", Dim, len(items)-endIndex, Reset)
			renderedLines++
		}

		fmt.Fprintf(out, "\r\n\r\n  %s↑/↓ Navigate · enter Select · esc Cancel%s", Dim, Reset)
		renderedLines += 2
	}

	render()

	for {
		ev, err := readKeyEvent(fd, in)
		if err != nil {
			clearMenu()
			return -1, err
		}

		switch ev.Type {
		case KeyUp:
			selectedIndex--
			if selectedIndex < 0 {
				selectedIndex = len(items) - 1
			}
			render()
		case KeyDown:
			selectedIndex++
			if selectedIndex >= len(items) {
				selectedIndex = 0
			}
			render()
		case KeyEnter:
			clearMenu()
			return selectedIndex, nil
		case KeyEscape, KeyCtrlC:
			clearMenu()
			return -1, nil
		case KeyRune:
			// If user pressed a digit 1-9
			if ev.Rune >= '1' && ev.Rune <= '9' {
				idx := int(ev.Rune - '1')
				if idx < len(items) {
					clearMenu()
					return idx, nil
				}
			}
		}
	}
}
