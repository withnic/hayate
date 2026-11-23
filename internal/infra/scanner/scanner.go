package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/withnic/hayate/internal/domain"
	"github.com/withnic/hayate/internal/infra/osenv"
)

type Scanner struct{}

func (Scanner) Scan(ctx context.Context, c domain.Config) []domain.ScanResult {
	return scan(ctx, c, scanRoot)
}
func scan(ctx context.Context, c domain.Config, walk func(context.Context, domain.RootConfig, []string) ([]domain.ProjectEntry, error)) []domain.ScanResult {
	results := make([]domain.ScanResult, len(c.Roots))
	var group sync.WaitGroup
	for i, r := range c.Roots {
		group.Add(1)
		go func() {
			defer group.Done()
			base := domain.ScanResult{Key: osenv.Key(r.Env, r.Distro, r.Path), Root: r}
			rootCtx, cancel := context.WithTimeout(ctx, time.Duration(c.ScanTimeoutMS)*time.Millisecond)
			defer cancel()
			ch := make(chan domain.ScanResult, 1)
			go func(result domain.ScanResult) {
				result.Entries, result.Err = walk(rootCtx, r, c.IgnoreDirs)
				ch <- result
			}(base)
			select {
			case result := <-ch:
				if err := rootCtx.Err(); err != nil {
					base.Err = err
				} else {
					base = result
				}
			case <-rootCtx.Done():
				base.Err = rootCtx.Err()
			}
			results[i] = base
		}()
	}
	group.Wait()
	return results
}

func scanRoot(ctx context.Context, r domain.RootConfig, ignore []string) ([]domain.ProjectEntry, error) {
	base, err := osenv.ScanPath(r)
	if err != nil {
		return nil, err
	}
	add := func(rel, scanPath string) domain.ProjectEntry {
		rel = filepath.ToSlash(rel)
		native := path.Join(r.Path, rel)
		if r.Env == domain.Windows {
			if osenv.LocalEnv() == domain.WSL {
				native = scanPath
			} else {
				native = filepath.Join(r.Path, rel)
			}
		}
		name := path.Base(rel)
		if rel == "." {
			name = path.Base(strings.ReplaceAll(r.Path, "\\", "/"))
		}
		if parent := path.Base(path.Dir(rel)); r.Mode == "ghq" && parent != "." {
			name = parent + "/" + name
		}
		return domain.ProjectEntry{ID: osenv.Key(r.Env, r.Distro, native), Env: r.Env, Distro: r.Distro, RootName: r.Name, DisplayName: name, RelPath: rel, NativePath: native, ScanPath: scanPath}
	}
	excluded := func(name string) bool { return strings.HasPrefix(name, ".") || slices.Contains(ignore, name) }
	entries := []domain.ProjectEntry{}
	if r.Mode == "children" {
		children, err := os.ReadDir(base)
		if err != nil {
			return nil, err
		}
		for _, d := range children {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if excluded(d.Name()) {
				continue
			}
			filename := filepath.Join(base, d.Name())
			isDir := d.IsDir()
			if !isDir && d.Type()&os.ModeSymlink != 0 {
				info, err := os.Stat(filename)
				isDir = err == nil && info.IsDir()
			}
			if isDir {
				entries = append(entries, add(d.Name(), filename))
			}
		}
		return entries, nil
	}
	info, err := os.Stat(base)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", base)
	}
	markers := r.Markers
	if r.Mode == "ghq" {
		markers = []string{".git"}
	}
	var walk func(string, string, int) error
	walk = func(filename, rel string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		children, err := os.ReadDir(filename)
		if err != nil {
			if depth != 0 && (errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist)) {
				return nil
			}
			return err
		}
		// ReadDir supplies both marker names and traversal candidates in one operation.
		for _, child := range children {
			if slices.Contains(markers, child.Name()) {
				entries = append(entries, add(rel, filename))
				return nil
			}
		}
		if r.Mode == "marker" && depth >= r.MaxDepth {
			return nil
		}
		for _, child := range children {
			if child.IsDir() && !excluded(child.Name()) {
				if err := walk(filepath.Join(filename, child.Name()), filepath.Join(rel, child.Name()), depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err = walk(base, ".", 0)
	return entries, err
}
