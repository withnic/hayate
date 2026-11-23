package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/withnic/hayate/internal/domain"
	"github.com/withnic/hayate/internal/infra/osenv"
)

func TestScanModes(t *testing.T) {
	base := t.TempDir()
	for _, rel := range []string{"host/owner/repo/sub/deep", "hidden/.git", "node_modules/skipped", ".secret"} {
		if err := os.MkdirAll(filepath.Join(base, rel), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{"host/owner/repo/.git", "host/owner/repo/sub/deep/go.mod", "host/owner/go.mod"} {
		if err := os.WriteFile(filepath.Join(base, rel), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	c := domain.DefaultConfig()
	for _, tt := range []struct {
		mode    string
		markers []string
		want    []string
	}{
		{"children", nil, []string{"hidden", "host"}},
		{"ghq", nil, []string{"hidden", "owner/repo"}},
		{"marker", []string{"go.mod"}, []string{"owner"}},
	} {
		c.Roots = []domain.RootConfig{{Path: base, Env: osenv.LocalEnv(), Distro: "Ubuntu", Mode: tt.mode, MaxDepth: 3, Markers: tt.markers}, {Path: filepath.Join(base, "missing"), Env: osenv.LocalEnv(), Distro: "Ubuntu", Mode: tt.mode}}
		res := Scanner{}.Scan(context.Background(), c)
		if res[0].Err != nil || !errors.Is(res[1].Err, os.ErrNotExist) || len(res[0].Entries) != len(tt.want) {
			t.Fatalf("%s: %+v", tt.mode, res)
		}
		for i, entry := range res[0].Entries {
			if entry.DisplayName != tt.want[i] {
				t.Errorf("%s: %q", tt.mode, entry.DisplayName)
			}
		}
	}
}
func TestScanTimeoutWhileWalkIsBlocked(t *testing.T) {
	c := domain.DefaultConfig()
	c.ScanTimeoutMS = 100
	c.Roots = []domain.RootConfig{{Path: "blocked"}, {Path: "ready"}}
	release := make(chan struct{})
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res := scan(ctx, c, func(ctx context.Context, r domain.RootConfig, _ []string) ([]domain.ProjectEntry, error) {
		if r.Path == "blocked" {
			<-release
		}
		return []domain.ProjectEntry{{ID: r.Path}}, nil
	})
	if !errors.Is(res[0].Err, context.DeadlineExceeded) || res[1].Err != nil || len(res[1].Entries) != 1 || ctx.Err() != nil {
		t.Fatalf("scan: %+v", res)
	}
}
