package config

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/withnic/hayate/internal/domain"
	"github.com/withnic/hayate/internal/infra/osenv"
)

func Resolve(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if configured := os.Getenv("HAYATE_CONFIG"); configured != "" {
		return configured, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hayate.jsonc"), nil
}
func Load(explicit string) (domain.Config, string, error) {
	cfg := domain.DefaultConfig()
	filename, err := Resolve(explicit)
	if err != nil {
		return cfg, "", err
	}
	raw, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, "", fmt.Errorf("%s: %s does not exist; run hayate init-config", domain.ConfigNotFound, filename)
	}
	if err != nil {
		return cfg, "", fmt.Errorf("%s: %w", domain.ConfigInvalid, err)
	}
	invalid := func(err error) (domain.Config, string, error) {
		return cfg, "", fmt.Errorf("%s: %s: %w", domain.ConfigInvalid, filename, err)
	}
	clean, err := jsonc(raw)
	if err != nil {
		return invalid(err)
	}
	var defaults struct{ Roots []json.RawMessage }
	if err = json.Unmarshal(clean, &cfg); err != nil {
		return invalid(err)
	}
	if err = json.Unmarshal(clean, &defaults); err != nil {
		return invalid(err)
	}
	for i, data := range defaults.Roots {
		root := domain.RootConfig{Mode: "children", MaxDepth: 3, Markers: []string{".git"}}
		if err = json.Unmarshal(data, &root); err != nil {
			return invalid(fmt.Errorf("roots[%d]: %w", i, err))
		}
		cfg.Roots[i] = root
	}
	if err = cfg.Validate(); err != nil {
		return invalid(err)
	}
	home, _ := os.UserHomeDir()
	for i, r := range cfg.Roots {
		cfg.Roots[i], err = osenv.NormalizeRoot(r, home)
		if err != nil {
			return invalid(fmt.Errorf("roots[%d].path: %w", i, err))
		}
	}
	return cfg, fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), nil
}
func Init(explicit string, force bool) error {
	filename, err := Resolve(explicit)
	if err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(filename, flags, 0600)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%s already exists; use --force to replace it", filename)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "{\n  // Change this to the parent directory of your projects.\n  \"roots\": [{\"env\": %q, \"path\": \"~/projects\", \"mode\": \"children\"}],\n  \"actions\": {\n    \"c\": {\"description\": \"Open VS Code\", \"windows\": \"code .\", \"wsl\": \"code .\"}\n  },\n}\n", osenv.LocalEnv())
	return errors.Join(err, f.Close())
}
