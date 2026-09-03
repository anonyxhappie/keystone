package ui

import (
	"io"
	"time"

	"golang.org/x/sys/unix"
)

// KeyType identifies the pressed key.
type KeyType int

const (
	KeyNone KeyType = iota
	KeyRune
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyEnter
	KeyTab
	KeyBackspace
	KeyDelete
	KeyEscape
	KeyCtrlC
	KeyCtrlD
)

// KeyEvent wraps the key type and optional rune value.
type KeyEvent struct {
	Type KeyType
	Rune rune
}

// readKeyEvent reads a single logical key event from raw terminal input.
func readKeyEvent(fd int, r io.Reader) (KeyEvent, error) {
	var b [1]byte
	n, err := r.Read(b[:])
	if err != nil {
		return KeyEvent{}, err
	}
	if n == 0 {
		return KeyEvent{Type: KeyNone}, nil
	}

	switch b[0] {
	case 3: // Ctrl+C
		return KeyEvent{Type: KeyCtrlC}, nil
	case 4: // Ctrl+D
		return KeyEvent{Type: KeyCtrlD}, nil
	case 9: // Tab
		return KeyEvent{Type: KeyTab}, nil
	case 10, 13: // Enter / Return
		return KeyEvent{Type: KeyEnter}, nil
	case 127, 8: // Backspace
		return KeyEvent{Type: KeyBackspace}, nil
	case 27: // Escape or start of ANSI escape sequence
		if fd >= 0 && !isDataAvailable(fd, 35*time.Millisecond) {
			return KeyEvent{Type: KeyEscape}, nil
		}
		var seq [2]byte
		n, err := r.Read(seq[:1])
		if err != nil || n == 0 {
			return KeyEvent{Type: KeyEscape}, nil
		}
		if seq[0] == '[' || seq[0] == 'O' {
			n, err := r.Read(seq[1:2])
			if err != nil || n == 0 {
				return KeyEvent{Type: KeyEscape}, nil
			}
			switch seq[1] {
			case 'A':
				return KeyEvent{Type: KeyUp}, nil
			case 'B':
				return KeyEvent{Type: KeyDown}, nil
			case 'C':
				return KeyEvent{Type: KeyRight}, nil
			case 'D':
				return KeyEvent{Type: KeyLeft}, nil
			case 'H':
				return KeyEvent{Type: KeyHome}, nil
			case 'F':
				return KeyEvent{Type: KeyEnd}, nil
			case '3': // \x1b[3~ is Delete key
				var tilda [1]byte
				_, _ = r.Read(tilda[:])
				return KeyEvent{Type: KeyDelete}, nil
			}
		}
		return KeyEvent{Type: KeyEscape}, nil
	default:
		if b[0] >= 32 {
			return KeyEvent{Type: KeyRune, Rune: rune(b[0])}, nil
		}
		return KeyEvent{Type: KeyNone}, nil
	}
}

func isDataAvailable(fd int, timeout time.Duration) bool {
	if fd < 0 {
		return false
	}
	fdset := &unix.FdSet{}
	fdset.Set(fd)
	tv := unix.NsecToTimeval(timeout.Nanoseconds())
	n, err := unix.Select(fd+1, fdset, nil, nil, &tv)
	return err == nil && n > 0
}
