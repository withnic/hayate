package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/withnic/hayate/internal/domain"
)

func TestFileCacheReplacesSnapshot(t *testing.T) {
	repo := FileCache{Path: filepath.Join(t.TempDir(), ".hayate_cache.json")}
	for _, hash := range []string{"sha256:old", "sha256:new"} {
		now := time.Now()
		data := domain.Cache{Version: 1, ConfigHash: hash, UpdatedAt: now, Entries: []domain.ProjectEntry{{ID: "project"}}}
		if err := repo.Save(data); err != nil {
			t.Fatal(err)
		}
		got, err := repo.Load()
		if err != nil || got.ConfigHash != hash || !got.UpdatedAt.Equal(now) || len(got.Entries) != 1 {
			t.Fatalf("cache: %+v %v", got, err)
		}
	}
}

func TestLoadFlatSnapshotRecoversRootKeys(t *testing.T) {
	repo := FileCache{Path: filepath.Join(t.TempDir(), ".hayate_cache.json")}
	old := domain.Cache{Version: 1, UpdatedAt: time.Now(), Entries: []domain.ProjectEntry{
		{Env: domain.WSL, Distro: "Ubuntu", NativePath: "/work/go/demo", RelPath: "go/demo"},
		{Env: domain.Windows, NativePath: `C:\Work\Demo`, RelPath: "Demo"},
	}}
	if err := repo.Save(old); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"wsl:Ubuntu:/work", "windows::c:/work"} {
		root := got.Roots[key]
		if !root.UpdatedAt.Equal(old.UpdatedAt) || len(root.Entries) != 1 {
			t.Errorf("root %q: %+v", key, root)
		}
	}
}
