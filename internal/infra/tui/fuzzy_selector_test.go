package tui

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/withnic/hayate/internal/domain"
)

func TestSelectProjectKeys(t *testing.T) {
	projects := []domain.ProjectEntry{
		{Env: domain.Windows, DisplayName: "api-client", RelPath: "a"},
		{Env: domain.WSL, Distro: "Ubuntu", DisplayName: "api-server", RelPath: "b"},
		{Env: domain.WSL, DisplayName: "docs", RelPath: "c"},
	}
	for _, tt := range []struct {
		name, query string
		keys        []rune
		want        int
		canceled    bool
		prompt      string
	}{
		{"initial query and move", "aPi", []rune{keyDown, '\r'}, 1, false, "aPi"},
		{"update query", "", []rune{'s', 'r', 'v', '\n'}, 1, false, "srv"},
		{"escape", "", []rune{keyCancel}, 0, true, "Type to search…"},
		{"control-c", "", []rune{3}, 0, true, "Type to search…"},
		{"no matches", "zzz", []rune{'\r', keyCancel}, 0, true, "zzz"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var screen bytes.Buffer
			keys := tt.keys
			tty := &terminal{Writer: &screen, readKey: func() (rune, error) {
				if len(keys) == 0 {
					return 0, io.EOF
				}
				key := keys[0]
				keys = keys[1:]
				return key, nil
			}}
			idx, err := selectProject(tty, projects, tt.query)
			if tt.canceled {
				if !errors.Is(err, domain.ErrCanceled) {
					t.Fatalf("cancel returned %v", err)
				}
			} else if err != nil || idx != tt.want {
				t.Fatalf("selected %d, %v", idx, err)
			}
			frames := strings.Split(screen.String(), "\x1b[H\x1b[2J")
			frame := frames[len(frames)-1]
			if !strings.HasPrefix(frame, "❯ "+tt.prompt+"\r\n\r\n") || !strings.HasSuffix(frame, "\x1b[?25h\x1b[?1049l") {
				t.Fatalf("terminal output %q", screen.String())
			}
			if tt.prompt == "srv" && (!strings.Contains(frame, "[WSL:Ubuntu] api-server") || strings.Contains(frame, "api-client") || !strings.Contains(frame, "1/3")) {
				t.Fatalf("filtered results %q", frame)
			}
			if tt.prompt == "zzz" && !strings.Contains(frame, "No matching projects") {
				t.Fatalf("empty results %q", frame)
			}
		})
	}
}
