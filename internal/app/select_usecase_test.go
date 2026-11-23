package app

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/withnic/hayate/internal/domain"
)

type scanFunc func(context.Context, domain.Config) []domain.ScanResult

func (f scanFunc) Scan(ctx context.Context, c domain.Config) []domain.ScanResult { return f(ctx, c) }

type selectFunc func([]domain.ProjectEntry, string) (int, error)

func (f selectFunc) Select(entries []domain.ProjectEntry, query string) (int, error) {
	return f(entries, query)
}

type memoryCache struct{ data domain.Cache }

func (m *memoryCache) Load() (domain.Cache, error)                 { return m.data, nil }
func (m *memoryCache) Save(c domain.Cache) error                   { m.data = c; return nil }
func identity(_ string, p domain.ProjectEntry) domain.ProjectEntry { return p }
func TestSelectOutputAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name, action, defaultAction, want string
		uiErr                             error
		errCode                           string
	}{
		{"move only", "", "", "cd '/dev/my project'\n", nil, ""},
		{"default", "", "code", "cd '/dev/my project' && code .\n", nil, ""},
		{"explicit overrides default", "other", "code", "cd '/dev/my project' && other .\n", nil, ""},
		{"canceled", "", "code", "", domain.ErrCanceled, ""},
		{"undefined action", "missing", "code", "", nil, domain.UnknownAction},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			called := false
			u := SelectUseCase{
				Load: func(string) (domain.Config, string, error) {
					return domain.Config{
						DefaultAction: test.defaultAction,
						Actions: map[string]domain.ActionConfig{
							"code": {WSL: "code ."}, "other": {WSL: "other ."},
						},
					}, "hash", nil
				},
				Scanner: scanFunc(func(context.Context, domain.Config) []domain.ScanResult {
					return []domain.ScanResult{{Entries: []domain.ProjectEntry{{ID: "dev", Env: domain.WSL, NativePath: "/dev/my project"}}}}
				}),
				UI: selectFunc(func(entries []domain.ProjectEntry, query string) (int, error) {
					called = true
					if query != "project" || len(entries) != 1 {
						t.Fatalf("selector input: %v %q", entries, query)
					}
					return 0, test.uiErr
				}),
				Out: &out, Err: &stderr, ForShell: identity,
			}
			err := u.Select(context.Background(), Options{Shell: "bash", Action: test.action, Query: "project"})
			if test.errCode != "" {
				if err == nil || !strings.Contains(err.Error(), test.errCode) || !strings.Contains(err.Error(), "code") || called {
					t.Fatalf("action validation: %v, called=%v", err, called)
				}
			} else if err != nil || !called {
				t.Fatalf("select: %v, called=%v", err, called)
			}
			if out.String() != test.want || stderr.Len() != 0 {
				t.Fatalf("stdout=%q stderr=%q", out.String(), stderr.String())
			}
		})
	}
}

func TestCacheInvalidation(t *testing.T) {
	c := domain.DefaultConfig()
	p := domain.ProjectEntry{ID: "a"}
	for _, tt := range []struct {
		name, hash   string
		updated      time.Time
		refresh, hit bool
	}{
		{"hit", "hash", time.Now(), false, true},
		{"expired", "hash", time.Now().Add(-time.Hour), false, false},
		{"changed", "old", time.Now(), false, false},
		{"refresh", "hash", time.Now(), true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			var stderr bytes.Buffer
			repo := &memoryCache{data: domain.Cache{Version: 1, ConfigHash: tt.hash, UpdatedAt: tt.updated, Entries: []domain.ProjectEntry{p}}}
			u := SelectUseCase{Cache: repo, Err: &stderr, Scanner: scanFunc(func(context.Context, domain.Config) []domain.ScanResult {
				calls++
				return []domain.ScanResult{{Key: "root", Entries: []domain.ProjectEntry{p}}}
			})}
			got, err := u.projects(context.Background(), c, "hash", tt.refresh)
			if err != nil || len(got) != 1 || (calls == 0) != tt.hit {
				t.Fatalf("entries=%+v calls=%d err=%v", got, calls, err)
			}
		})
	}
}

func TestPartialScanUsesOnlyMatchingTimedOutCache(t *testing.T) {
	c := domain.DefaultConfig()
	oldTime := time.Now().Add(-time.Hour)
	old := domain.ProjectEntry{ID: "old", DisplayName: "alpha"}
	recent := domain.ProjectEntry{ID: "new", DisplayName: "beta"}
	var stderr bytes.Buffer
	repo := &memoryCache{data: domain.Cache{
		Version: 1, ConfigHash: "old hash", UpdatedAt: oldTime,
		Roots: map[string]domain.CachedRoot{
			"wsl:Ubuntu:/work":  {UpdatedAt: oldTime, Entries: []domain.ProjectEntry{old}},
			"wsl:Ubuntu:/other": {UpdatedAt: oldTime, Entries: []domain.ProjectEntry{{ID: "other"}}},
		},
	}}
	u := SelectUseCase{Cache: repo, Err: &stderr, Scanner: scanFunc(func(context.Context, domain.Config) []domain.ScanResult {
		return []domain.ScanResult{
			{Key: "wsl:Ubuntu:/work", Root: domain.RootConfig{Path: "/work"}, Err: context.DeadlineExceeded},
			{Key: "wsl:Ubuntu:/other", Root: domain.RootConfig{Path: "/other"}, Err: errors.New("inaccessible")},
			{Key: "fresh", Entries: []domain.ProjectEntry{recent, old}},
		}
	})}
	got, err := u.projects(context.Background(), c, "hash", false)
	if err != nil || len(got) != 2 || got[0].ID != "old" || got[1].ID != "new" {
		t.Fatalf("entries=%+v err=%v", got, err)
	}
	if strings.Count(stderr.String(), domain.ScanPartialFail) != 2 || !repo.data.Roots["wsl:Ubuntu:/work"].UpdatedAt.Equal(oldTime) || !repo.data.UpdatedAt.IsZero() {
		t.Fatalf("stderr=%q cache=%+v", stderr.String(), repo.data)
	}
}
