package cli

import (
	"fmt"
	"strings"

	"github.com/withnic/hayate/internal/app"
)

type options struct {
	app.Options
	command, alias    string
	help, force, json bool
}

func parse(args []string) (options, error) {
	o := options{command: "select"}
	positional := []string{}
	seen := map[string]bool{}
	explicit := false
	flags := true
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if flags && arg == "--" {
			flags = false
			continue
		}
		if !flags || arg == "-" || !strings.HasPrefix(arg, "-") {
			if !explicit && flags && len(positional) == 0 {
				switch arg {
				case "select", "list", "init", "init-config", "version":
					o.command = arg
					explicit = true
					continue
				}
			}
			positional = append(positional, arg)
			continue
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-"), "=")
		switch name {
		case "config", "shell", "cmd", "action", "query", "q":
			if !hasValue {
				i++
				if i == len(args) {
					return o, fmt.Errorf("%s needs a value", arg)
				}
				value = args[i]
			}
			switch name {
			case "config":
				o.Config = value
			case "shell":
				o.Shell = value
			case "cmd":
				o.alias = value
			case "action":
				o.Action = value
			case "q", "query":
				o.Query = value
				name = "query"
			}
		case "refresh", "json", "force", "h", "help", "version":
			if hasValue {
				return o, fmt.Errorf("%s does not take a value", name)
			}
			switch name {
			case "refresh":
				o.Refresh = true
			case "json":
				o.json = true
			case "force":
				o.force = true
			case "h", "help":
				o.help = true
			case "version":
				o.command = "version"
				explicit = true
			}
		default:
			if name == "" || hasValue {
				return o, fmt.Errorf("invalid flag %q", arg)
			}
			o.Action = name
			name = "action"
		}
		seen[name] = true
	}
	if o.help {
		return o, nil
	}
	var allowed string
	switch o.command {
	case "select":
		allowed = "config shell query action refresh"
		if len(positional) == 1 && !seen["query"] {
			o.Query = positional[0]
		}
	case "list":
		allowed = "config refresh json"
	case "init":
		allowed = "cmd"
		if len(positional) != 1 {
			return o, fmt.Errorf("usage: hayate init <powershell|bash|zsh> [--cmd <name>]")
		}
		o.Shell = positional[0]
	case "init-config":
		allowed = "config force"
	case "version":
		allowed = "version"
	}
	for _, name := range strings.Fields(allowed) {
		delete(seen, name)
	}
	if len(seen) > 0 {
		return o, fmt.Errorf("unsupported flag for hayate %s", o.command)
	}
	if (o.command == "init" || o.command == "select") && len(positional) <= 1 || len(positional) == 0 {
		return o, nil
	}
	return o, fmt.Errorf("unexpected positional arguments for hayate %s", o.command)
}
