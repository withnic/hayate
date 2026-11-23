package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/withnic/hayate/internal/domain"
	"github.com/withnic/hayate/internal/infra/osenv"
)

type FileCache struct{ Path string }

func (c FileCache) Load() (domain.Cache, error) {
	var data domain.Cache
	raw, err := os.ReadFile(c.Path)
	if err != nil {
		return data, err
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		return data, err
	}
	if data.Version != 1 {
		return data, fmt.Errorf("unsupported cache version %d", data.Version)
	}
	if data.Roots == nil {
		data.Roots = recoverRoots(data)
	}
	return data, nil
}

// Version 1 snapshots can contain only a flat entries list.
func recoverRoots(data domain.Cache) map[string]domain.CachedRoot {
	roots := make(map[string]domain.CachedRoot)
	for _, entry := range data.Entries {
		key := osenv.Key(entry.Env, entry.Distro, entry.NativePath)
		if entry.RelPath != "." {
			rel := entry.RelPath
			if entry.Env == domain.Windows {
				rel = strings.ToLower(strings.ReplaceAll(rel, "\\", "/"))
			}
			if rel == "" || !strings.HasSuffix(key, "/"+rel) {
				continue
			}
			key = strings.TrimSuffix(key, "/"+rel)
			if strings.HasSuffix(key, ":") {
				key += "/"
			}
		}
		root := roots[key]
		root.UpdatedAt = data.UpdatedAt
		root.Entries = append(root.Entries, entry)
		roots[key] = root
	}
	return roots
}

func (c FileCache) Save(data domain.Cache) error {
	if c.Path == "" {
		return fmt.Errorf("home directory is unavailable")
	}
	f, err := os.CreateTemp(filepath.Dir(c.Path), ".hayate_cache_*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = json.NewEncoder(f).Encode(data)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), c.Path)
}
