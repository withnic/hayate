package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/withnic/hayate/internal/app"
)

type Runner struct {
	UseCase      app.SelectUseCase
	InitConfig   func(string, bool) error
	DefaultShell func() string
	Version      string
	Out, Err     io.Writer
}

func (r Runner) Run(ctx context.Context, args []string) int {
	if err := r.run(ctx, args); err != nil {
		fmt.Fprintln(r.Err, err)
		return 1
	}
	return 0
}
func (r Runner) run(ctx context.Context, args []string) error {
	o, err := parse(args)
	if err != nil {
		return err
	}
	if o.help {
		_, err = fmt.Fprintln(r.Err, "Usage: hayate [select] [--config <path>] [--shell <shell>] [--action <name> | -<name>] [--refresh] [--query <text> | <initial_query>]\n       hayate init <powershell|bash|zsh> [--cmd <name>]\n       hayate init-config [--force]\n       hayate list [--json] [--refresh]\n       hayate version")
		if c, _, loadErr := r.UseCase.Load(o.Config); loadErr == nil {
			for _, name := range app.ActionNames(c) {
				fmt.Fprintf(r.Err, "  -%s  %s\n", name, c.Actions[name].Description)
			}
		}
		return err
	}
	switch o.command {
	case "init":
		return app.Init(r.Out, o.Shell, o.alias)
	case "init-config":
		return r.InitConfig(o.Config, o.force)
	case "list":
		return r.UseCase.List(ctx, o.Options, o.json)
	case "version":
		_, err = fmt.Fprintln(r.Out, "hayate "+r.Version)
		return err
	}
	if o.Shell == "" {
		o.Shell = r.DefaultShell()
	}
	switch o.Shell {
	case "powershell", "powershell-legacy", "bash", "zsh":
		return r.UseCase.Select(ctx, o.Options)
	default:
		return fmt.Errorf("unsupported shell %q", o.Shell)
	}
}
