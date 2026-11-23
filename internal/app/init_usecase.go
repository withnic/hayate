package app

import (
	"fmt"
	"io"
	"regexp"
)

var aliasPattern = regexp.MustCompile("^[A-Za-z_][A-Za-z0-9_-]*$")

func Init(out io.Writer, shell, name string) error {
	if name == "" {
		name = "j"
	}
	if !aliasPattern.MatchString(name) {
		return fmt.Errorf("invalid wrapper name %q", name)
	}
	var script string
	switch shell {
	case "powershell":
		script = `function %s {
    $__hayate_shell = 'powershell'
    if ($PSVersionTable.PSVersion.Major -lt 7) {
        $__hayate_shell = 'powershell-legacy'
    }
    $__hayate_cmd = (& hayate.exe select --shell $__hayate_shell @args) -join "%sn"
    if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($__hayate_cmd)) {
        Invoke-Expression $__hayate_cmd
    }
}
`
		_, err := fmt.Fprintf(out, script, name, string(rune(96)))
		return err
	case "bash", "zsh":
		script = `%s() {
    local __hayate_cmd
    __hayate_cmd="$(hayate select --shell %s "$@")" || return $?
    if [ -n "$__hayate_cmd" ]; then
        eval "$__hayate_cmd"
    fi
}
`
		_, err := fmt.Fprintf(out, script, name, shell)
		return err
	default:
		return fmt.Errorf("unsupported shell %q (use powershell, bash or zsh)", shell)
	}
}
