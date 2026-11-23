package osenv

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/withnic/hayate/internal/domain"
)

func LocalEnv() domain.EnvType {
	if runtime.GOOS == "windows" {
		return domain.Windows
	}
	return domain.WSL
}
func Shell() string {
	if LocalEnv() == domain.Windows {
		return "powershell"
	}
	if path.Base(os.Getenv("SHELL")) == "zsh" {
		return "zsh"
	}
	return "bash"
}
func WindowsToWSL(p string) (string, bool) {
	p = strings.ReplaceAll(p, "\\", "/")
	if len(p) < 3 || p[1:3] != ":/" || !strings.Contains("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", p[:1]) {
		return "", false
	}
	return path.Clean("/mnt/" + strings.ToLower(p[:1]) + p[2:]), true
}
func WSLToWindows(p string) (string, bool) {
	if len(p) < 6 || p[:5] != "/mnt/" || !strings.Contains("abcdefghijklmnopqrstuvwxyz", p[5:6]) || (len(p) > 6 && p[6] != '/') {
		return "", false
	}
	return strings.ToUpper(p[5:6]) + ":\\" + strings.ReplaceAll(strings.TrimPrefix(p[6:], "/"), "/", "\\"), true
}
func WSLToUNC(distro, p string) string {
	return "\\\\" + "wsl.localhost" + "\\" + distro + "\\" + strings.ReplaceAll(strings.TrimPrefix(path.Clean(p), "/"), "/", "\\")
}
func NormalizeRoot(r domain.RootConfig, home string) (domain.RootConfig, error) {
	if r.Env == domain.Windows {
		r.Distro = ""
	}
	if r.Env == domain.WSL && r.Distro == "" {
		r.Distro = "Ubuntu"
	}
	if r.Path == "~" || strings.HasPrefix(r.Path, "~/") || (r.Env == domain.Windows && strings.HasPrefix(r.Path, "~\\")) {
		if r.Env != LocalEnv() {
			return r, fmt.Errorf("use an absolute %s path instead of ~ (for WSL, /home/<user>/…)", r.Env)
		}
		if home == "" {
			return r, fmt.Errorf("home directory is unavailable")
		}
		r.Path = filepath.Join(home, r.Path[1:])
	}
	switch r.Env {
	case domain.WSL:
		if strings.HasPrefix(r.Path, "/") {
			r.Path = path.Clean(r.Path)
			return r, nil
		}
	case domain.Windows:
		if LocalEnv() == domain.Windows && filepath.IsAbs(r.Path) {
			r.Path = filepath.Clean(r.Path)
			return r, nil
		}
		if _, ok := WindowsToWSL(r.Path); ok {
			p := strings.ReplaceAll(r.Path, "\\", "/")
			r.Path = p[:2] + strings.ReplaceAll(path.Clean(p[2:]), "/", "\\")
			return r, nil
		}
	}
	return r, fmt.Errorf("use an absolute %s path", r.Env)
}
func ScanPath(r domain.RootConfig) (string, error) {
	if LocalEnv() == domain.Windows && r.Env == domain.WSL {
		return WSLToUNC(r.Distro, r.Path), nil
	}
	if LocalEnv() == domain.WSL && r.Env == domain.Windows {
		if p, ok := WindowsToWSL(r.Path); ok {
			return p, nil
		}
		return "", fmt.Errorf("Windows drive path required")
	}
	if r.Env == domain.WSL {
		if current := os.Getenv("WSL_DISTRO_NAME"); current != "" && !strings.EqualFold(current, r.Distro) {
			return "", fmt.Errorf("distribution %s is not mounted in %s", r.Distro, current)
		}
	}
	return r.Path, nil
}
func Key(env domain.EnvType, distro, p string) string {
	if env == domain.Windows {
		if windows, ok := WSLToWindows(p); ok {
			p = windows
		}
		p = strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
		distro = ""
	}
	return string(env) + ":" + distro + ":" + p
}
func ForShell(shell string, p domain.ProjectEntry) domain.ProjectEntry {
	if p.Env == domain.Windows {
		if shell == "powershell" || shell == "powershell-legacy" {
			if converted, ok := WSLToWindows(p.NativePath); ok {
				p.NativePath = converted
			}
		} else if converted, ok := WindowsToWSL(p.NativePath); ok {
			p.NativePath = converted
		}
	}
	return p
}
