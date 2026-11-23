package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONC(t *testing.T) {
	raw := []byte("\xef\xbb\xbf" + "{\"windows\":\"C:\\\\Users\\\\foo\\\\\", /* block */\n\"url\":\"http://example.com\", \"literal\": \"/* not comment */\", // trailing\n}")
	clean, err := jsonc(raw)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err = json.Unmarshal(clean, &got); err != nil {
		t.Fatal(err)
	}
	if got["windows"] != "C:\\Users\\foo\\" || got["url"] != "http://example.com" || got["literal"] != "/* not comment */" {
		t.Fatalf("decoded %#v", got)
	}
	for _, input := range []string{"{/*", "{,}", "[,]", "[1,,]", "[1/* comment */2]"} {
		if _, err := jsonc([]byte(input)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}
func TestLoadDefaultsValidationAndHash(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hayate.jsonc")
	write := func(raw string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root := "{\"roots\":[{\"env\":\"wsl\",\"path\":\"/home/dev\",\"mode\":\"marker\"}]"
	write(root + "}")
	cfg, hash, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheTTLSeconds != 300 || cfg.Roots[0].Distro != "Ubuntu" || cfg.Roots[0].MaxDepth != 3 || cfg.Roots[0].Markers[0] != ".git" {
		t.Fatalf("defaults: %+v", cfg)
	}
	write(root + ",\"cache_ttl_seconds\":0}")
	cfg, next, err := Load(p)
	if err != nil || cfg.CacheTTLSeconds != 0 || next == hash {
		t.Fatalf("explicit TTL/hash: %+v %s %v", cfg, next, err)
	}
	write("{\"roots\":[]}")
	if _, _, err = Load(p); err == nil || !strings.Contains(err.Error(), "ERR_1002") {
		t.Fatalf("validation: %v", err)
	}
	write(root + ",\"default_action\":\"c\",\"actions\":{\"c\":{\"wsl\":\"code .\"}}}")
	cfg, _, err = Load(p)
	if err != nil || cfg.DefaultAction != "c" {
		t.Fatalf("default action: %q, %v", cfg.DefaultAction, err)
	}
	write(root + ",\"default_action\":\"missing\"}")
	if _, _, err = Load(p); err == nil || !strings.Contains(err.Error(), "ERR_1002") || !strings.Contains(err.Error(), "default_action") {
		t.Fatalf("undefined default action: %v", err)
	}
}
