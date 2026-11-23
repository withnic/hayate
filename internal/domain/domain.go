package domain

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

type EnvType string

const (
	Windows EnvType = "windows"
	WSL     EnvType = "wsl"
)
const (
	ConfigNotFound  = "ERR_1001"
	ConfigInvalid   = "ERR_1002"
	UnknownAction   = "ERR_1003"
	ScanPartialFail = "ERR_2001"
	NoProjectsFound = "ERR_2002"
	TTYUnavailable  = "ERR_3001"
)

var ErrCanceled = errors.New("selection canceled")

type RootConfig struct {
	Name     string
	Env      EnvType
	Distro   string
	Path     string
	Mode     string
	Markers  []string
	MaxDepth int `json:"max_depth"`
}
type ActionConfig struct{ Description, Windows, WSL string }
type Config struct {
	Roots           []RootConfig
	Actions         map[string]ActionConfig
	DefaultAction   string   `json:"default_action"`
	CacheTTLSeconds int      `json:"cache_ttl_seconds"`
	ScanTimeoutMS   int      `json:"scan_timeout_ms"`
	IgnoreDirs      []string `json:"ignore_dirs"`
}

func DefaultConfig() Config {
	return Config{CacheTTLSeconds: 300, ScanTimeoutMS: 3000, IgnoreDirs: []string{".git", "node_modules", "vendor", ".venv", "__pycache__"}}
}

var actionName = regexp.MustCompile("^[A-Za-z0-9_][A-Za-z0-9_-]*$")

func (c Config) Validate() error {
	if len(c.Roots) == 0 {
		return fmt.Errorf("roots must contain at least one directory")
	}
	if c.CacheTTLSeconds < 0 {
		return fmt.Errorf("cache_ttl_seconds must be nonnegative")
	}
	if c.ScanTimeoutMS < 100 || c.ScanTimeoutMS > 60000 {
		return fmt.Errorf("scan_timeout_ms must be 100–60000")
	}
	for i, r := range c.Roots {
		var message string
		switch {
		case r.Env != Windows && r.Env != WSL:
			message = "env must be windows or wsl"
		case strings.TrimSpace(r.Path) == "" || HasControl(r.Path):
			message = "path must be an absolute directory"
		case HasControl(r.Distro) || strings.ContainsAny(r.Distro, "/\\:") || r.Distro == "." || r.Distro == "..":
			message = "distro must be a distribution name"
		case r.Mode != "children" && r.Mode != "ghq" && r.Mode != "marker":
			message = "mode must be children, ghq or marker"
		case r.Mode == "marker" && (r.MaxDepth < 1 || r.MaxDepth > 10):
			message = "max_depth must be 1–10"
		}
		if message != "" {
			return fmt.Errorf("roots[%d].%s", i, message)
		}
		for _, m := range r.Markers {
			if m == "" || m == "." || m == ".." || HasControl(m) || strings.ContainsAny(m, "/\\:") {
				return fmt.Errorf("roots[%d].markers must contain file or directory names", i)
			}
		}
	}
	for name := range c.Actions {
		if !actionName.MatchString(name) || slices.Contains([]string{"config", "shell", "action", "refresh", "query", "q", "cmd", "force", "json", "help", "h", "version"}, name) {
			return fmt.Errorf("actions.%s is not an available action name", name)
		}
	}
	if _, ok := c.Actions[c.DefaultAction]; c.DefaultAction != "" && !ok {
		return fmt.Errorf("default_action %q must be defined in actions", c.DefaultAction)
	}
	return nil
}

type ProjectEntry struct {
	ID          string  `json:"id"`
	Env         EnvType `json:"env"`
	Distro      string  `json:"distro"`
	RootName    string  `json:"root_name"`
	DisplayName string  `json:"display_name"`
	RelPath     string  `json:"rel_path"`
	NativePath  string  `json:"native_path"`
	ScanPath    string  `json:"scan_path"`
}
type CachedRoot struct {
	UpdatedAt time.Time      `json:"updated_at"`
	Entries   []ProjectEntry `json:"entries"`
}
type Cache struct {
	Version    int                   `json:"version"`
	ConfigHash string                `json:"config_hash"`
	UpdatedAt  time.Time             `json:"updated_at"`
	Entries    []ProjectEntry        `json:"entries"`
	Roots      map[string]CachedRoot `json:"roots,omitempty"`
}
type ScanResult struct {
	Key     string
	Root    RootConfig
	Entries []ProjectEntry
	Err     error
}
type Scanner interface {
	Scan(context.Context, Config) []ScanResult
}
type CacheRepository interface {
	Load() (Cache, error)
	Save(Cache) error
}
type SelectorUI interface {
	Select([]ProjectEntry, string) (int, error)
}

func HasControl(s string) bool { return strings.ContainsAny(s, "\r\n\x00") }
