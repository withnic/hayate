package domain

import (
	"fmt"
	"strings"
)

var psQuote = strings.NewReplacer("'", "''", "‘", "‘‘", "’", "’’", "‚", "‚‚", "‛", "‛‛")

func BuildCommand(shell string, p ProjectEntry, action ActionConfig) (string, error) {
	if p.NativePath == "" || HasControl(p.NativePath) || HasControl(p.Distro) {
		return "", fmt.Errorf("invalid project path or distribution name")
	}
	var command, follow string
	switch shell {
	case "powershell", "powershell-legacy":
		quote := func(s string) string { return "'" + psQuote.Replace(s) + "'" }
		if p.Env == WSL {
			distro := p.Distro
			if distro == "" {
				distro = "Ubuntu"
			}
			command = "wsl.exe -d " + quote(distro) + " --cd " + quote(p.NativePath)
			if strings.TrimSpace(action.WSL) != "" {
				command += " -- bash -lc " + quote(action.WSL+"; exec \""+"$"+"{SHELL:-bash}\"")
			}
			return command, nil
		}
		command = "Set-Location -LiteralPath " + quote(p.NativePath)
		follow = action.Windows
	case "bash", "zsh":
		command = "cd '" + strings.ReplaceAll(p.NativePath, "'", "'\\''") + "'"
		follow = action.WSL
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
	if strings.TrimSpace(follow) != "" {
		if shell == "powershell-legacy" {
			return command + "; if ($?) {\n" + follow + "\n}", nil
		}
		command += " && " + follow
	}
	return command, nil
}
