package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/withnic/hayate/internal/domain"
)

type SelectUseCase struct {
	Load     func(string) (domain.Config, string, error)
	Scanner  domain.Scanner
	Cache    domain.CacheRepository
	UI       domain.SelectorUI
	ForShell func(string, domain.ProjectEntry) domain.ProjectEntry
	Out, Err io.Writer
}
type Options struct {
	Config, Shell, Action, Query string
	Refresh                      bool
}

func ActionNames(c domain.Config) []string {
	names := make([]string, 0, len(c.Actions))
	for n := range c.Actions {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
func (u SelectUseCase) Select(ctx context.Context, o Options) error {
	c, hash, err := u.Load(o.Config)
	if err != nil {
		return err
	}
	if o.Action == "" {
		o.Action = c.DefaultAction
	}
	action, ok := c.Actions[o.Action]
	if o.Action != "" && !ok {
		names := strings.Join(ActionNames(c), ", ")
		if names == "" {
			names = "(none)"
		}
		return fmt.Errorf("%s: unknown action %q; configured actions: %s", domain.UnknownAction, o.Action, names)
	}
	entries, err := u.projects(ctx, c, hash, o.Refresh)
	if err != nil {
		return err
	}
	index, err := u.UI.Select(entries, o.Query)
	if errors.Is(err, domain.ErrCanceled) {
		return nil
	}
	if err != nil {
		return err
	}
	if index < 0 || index >= len(entries) {
		return fmt.Errorf("invalid selection index %d", index)
	}
	cmd, err := domain.BuildCommand(o.Shell, u.ForShell(o.Shell, entries[index]), action)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(u.Out, cmd)
	return err
}
func (u SelectUseCase) List(ctx context.Context, o Options, asJSON bool) error {
	c, hash, err := u.Load(o.Config)
	if err != nil {
		return err
	}
	entries, err := u.projects(ctx, c, hash, o.Refresh)
	if err != nil {
		return err
	}
	if asJSON {
		return json.NewEncoder(u.Out).Encode(entries)
	}
	escape := strings.NewReplacer("\t", "\\t", "\r", "\\r", "\n", "\\n")
	for _, p := range entries {
		if _, err := fmt.Fprintf(u.Out, "%s\t%s\t%s\n", p.Env, escape.Replace(p.DisplayName), escape.Replace(p.NativePath)); err != nil {
			return err
		}
	}
	return nil
}
func fresh(now, updated time.Time, ttl int) bool {
	return !updated.IsZero() && !updated.After(now) && now.Sub(updated).Seconds() <= float64(ttl)
}
func (u SelectUseCase) projects(ctx context.Context, c domain.Config, hash string, refresh bool) ([]domain.ProjectEntry, error) {
	var old domain.Cache
	now := time.Now()
	if c.CacheTTLSeconds > 0 {
		if loaded, err := u.Cache.Load(); err == nil {
			old = loaded
		}
		if !refresh && old.ConfigHash == hash && len(old.Entries) > 0 && fresh(now, old.UpdatedAt, c.CacheTTLSeconds) {
			allFresh := true
			for _, root := range old.Roots {
				if !fresh(now, root.UpdatedAt, c.CacheTTLSeconds) {
					allFresh = false
					break
				}
			}
			if allFresh {
				return old.Entries, nil
			}
		}
	}
	roots := make(map[string]domain.CachedRoot, len(c.Roots))
	entries := make([]domain.ProjectEntry, 0)
	seen := map[string]bool{}
	partial := false
	for _, res := range u.Scanner.Scan(ctx, c) {
		current := domain.CachedRoot{UpdatedAt: time.Now(), Entries: res.Entries}
		if res.Err != nil {
			partial = true
			current = old.Roots[res.Key]
			fmt.Fprintf(u.Err, "%s: %s: %v\n", domain.ScanPartialFail, res.Root.Path, res.Err)
			if !errors.Is(res.Err, context.DeadlineExceeded) {
				roots[res.Key] = current
				continue
			}
		}
		roots[res.Key] = current
		for _, p := range current.Entries {
			if !seen[p.ID] {
				seen[p.ID] = true
				entries = append(entries, p)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		paths := make([]string, 0, len(c.Roots))
		for _, r := range c.Roots {
			paths = append(paths, r.Path)
		}
		return nil, fmt.Errorf("%s: no projects found in %s", domain.NoProjectsFound, strings.Join(paths, ", "))
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.DisplayName != b.DisplayName {
			return a.DisplayName < b.DisplayName
		}
		return a.ID < b.ID
	})
	if c.CacheTTLSeconds > 0 {
		data := domain.Cache{Version: 1, ConfigHash: hash, Entries: entries, Roots: roots}
		if !partial {
			data.UpdatedAt = time.Now()
		}
		if err := u.Cache.Save(data); err != nil {
			fmt.Fprintf(u.Err, "cache: %v\n", err)
		}
	}
	return entries, nil
}
