package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/withnic/hayate/internal/app"
	"github.com/withnic/hayate/internal/domain"
)

func TestParse(t *testing.T) {
	for _, test := range []struct {
		args                   []string
		command, action, query string
		refresh                bool
	}{
		{[]string{"-zed"}, "select", "zed", "", false},
		{[]string{"--claude"}, "select", "claude", "", false},
		{[]string{"--refresh", "-zed", "initial"}, "select", "zed", "initial", true},
		{[]string{"select", "--query=explicit", "position"}, "select", "", "explicit", false},
		{[]string{"--config", "custom", "list", "--json"}, "list", "", "", false},
	} {
		got, err := parse(test.args)
		if err != nil || got.command != test.command || got.Action != test.action || got.Query != test.query || got.Refresh != test.refresh {
			t.Errorf("%v: %+v %v", test.args, got, err)
		}
	}
}
func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runner := Runner{Out: &stdout, Err: &stderr, UseCase: app.SelectUseCase{Load: func(string) (domain.Config, string, error) {
		return domain.Config{Actions: map[string]domain.ActionConfig{"zed": {Description: "Open Zed"}}}, "", nil
	}}}
	if code := runner.Run(context.Background(), []string{"--help"}); code != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "Open Zed") {
		t.Fatalf("help: %d stdout=%q stderr=%q", code, &stdout, &stderr)
	}
}
