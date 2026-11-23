package domain

import "testing"

func TestBuildCommand(t *testing.T) {
	action := ActionConfig{Windows: "code .", WSL: "printf '%s' ok"}
	tests := []struct {
		shell  string
		p      ProjectEntry
		action ActionConfig
		want   string
	}{
		{"powershell", ProjectEntry{Env: Windows, NativePath: `C:\my work\[id]'s`}, action, `Set-Location -LiteralPath 'C:\my work\[id]''s' && code .`},
		{"powershell-legacy", ProjectEntry{Env: Windows, NativePath: `C:\my work\[id]'s`}, action, "Set-Location -LiteralPath 'C:\\my work\\[id]''s'; if ($?) {\ncode .\n}"},
		{"bash", ProjectEntry{Env: Windows, NativePath: "/mnt/c/my work/[id]'s"}, action, `cd '/mnt/c/my work/[id]'\''s' && printf '%s' ok`},
		{"zsh", ProjectEntry{Env: WSL, NativePath: "/home/dev"}, ActionConfig{}, "cd '/home/dev'"},
		{"powershell", ProjectEntry{Env: WSL, Distro: "Ubuntu", NativePath: "/home/dev"}, ActionConfig{}, "wsl.exe -d 'Ubuntu' --cd '/home/dev'"},
		{"powershell", ProjectEntry{Env: WSL, Distro: "Ubuntu", NativePath: "/home/dev"}, action, `wsl.exe -d 'Ubuntu' --cd '/home/dev' -- bash -lc 'printf ''%s'' ok; exec "` + "$" + `{SHELL:-bash}"'`},
	}
	for _, tt := range tests {
		got, err := BuildCommand(tt.shell, tt.p, tt.action)
		if err != nil || got != tt.want {
			t.Errorf("BuildCommand(%s, %q) = %q, %v; want %q", tt.shell, tt.p.NativePath, got, err, tt.want)
		}
	}
	for _, bad := range []ProjectEntry{{NativePath: "/tmp/bad\npath"}, {NativePath: "/tmp", Distro: "bad\x00distro"}, {NativePath: "/tmp\r"}} {
		if _, err := BuildCommand("bash", bad, action); err == nil {
			t.Errorf("accepted %#v", bad)
		}
	}
}
