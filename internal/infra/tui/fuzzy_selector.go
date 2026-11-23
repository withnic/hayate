package tui

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/withnic/hayate/internal/domain"
)

type FuzzySelector struct{}

const (
	keyUp rune = -1 - iota
	keyDown
	keyCancel
	keyIgnore
)

type terminal struct {
	io.Writer
	readKey func() (rune, error)
	close   func()
}

func (FuzzySelector) Select(projects []domain.ProjectEntry, initial string) (int, error) {
	tty, err := openTerminal()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", domain.TTYUnavailable, err)
	}
	defer tty.close()
	return selectProject(tty, projects, initial)
}

func selectProject(tty *terminal, projects []domain.ProjectEntry, initial string) (int, error) {
	var err error
	if _, err = fmt.Fprint(tty, "\x1b[?1049h\x1b[?25l"); err != nil {
		return 0, err
	}
	defer fmt.Fprint(tty, "\x1b[?25h\x1b[?1049l")

	labels := make([]string, len(projects))
	searchable := make([]string, len(projects))
	for i, p := range projects {
		badge := "[WIN]"
		if p.Env == domain.WSL {
			badge = "[WSL:" + p.Distro + "]"
		}
		label := badge + " " + p.DisplayName + "  " + p.RelPath
		if p.RootName != "" {
			label += "  (" + p.RootName + ")"
		}
		labels[i] = strings.Map(printable, label)
		searchable[i] = strings.ToLower(labels[i])
	}
	query := []rune(strings.Map(printable, initial))
	selected := 0
	for {
		matches := make([]int, 0, len(projects))
		terms := strings.Fields(strings.ToLower(string(query)))
		for i, label := range searchable {
			if matchesTerms(label, terms) {
				matches = append(matches, i)
			}
		}
		if selected >= len(matches) {
			selected = max(0, len(matches)-1)
		}
		var screen strings.Builder
		screen.WriteString("\x1b[H\x1b[2J❯ ")
		if len(query) == 0 {
			screen.WriteString("Type to search…")
		} else {
			screen.WriteString(string(query))
		}
		screen.WriteString("\r\n\r\n")
		if len(matches) == 0 {
			screen.WriteString("  No matching projects\r\n")
		}
		start := max(0, selected-7)
		for i := start; i < min(start+8, len(matches)); i++ {
			if i == selected {
				fmt.Fprintf(&screen, "  \x1b[7m%s\x1b[0m\r\n", labels[matches[i]])
			} else {
				fmt.Fprintf(&screen, "  %s\r\n", labels[matches[i]])
			}
		}
		fmt.Fprintf(&screen, "\r\n%d/%d  ↑↓ Move · Enter Select · Esc Cancel\r\n", len(matches), len(projects))
		if _, err = io.WriteString(tty, screen.String()); err != nil {
			return 0, err
		}
		key, err := tty.readKey()
		if err != nil {
			return 0, err
		}
		switch key {
		case keyCancel, 3, 27:
			return 0, domain.ErrCanceled
		case '\r', '\n':
			if len(matches) > 0 {
				return matches[selected], nil
			}
		case keyUp, 16:
			selected = max(0, selected-1)
		case keyDown, 14:
			selected = min(selected+1, max(0, len(matches)-1))
		case 8, 127:
			if len(query) > 0 {
				query = query[:len(query)-1]
				selected = 0
			}
		case 21:
			query = nil
			selected = 0
		default:
			if unicode.IsPrint(key) {
				query = append(query, key)
				selected = 0
			}
		}
	}
}
func printable(r rune) rune {
	if unicode.IsPrint(r) {
		return r
	}
	return ' '
}
func matchesTerms(label string, terms []string) bool {
	for _, term := range terms {
		remaining := label
		for _, r := range term {
			_, tail, found := strings.Cut(remaining, string(r))
			if !found {
				return false
			}
			remaining = tail
		}
	}
	return true
}
