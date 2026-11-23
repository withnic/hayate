package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/withnic/hayate/internal/app"
	"github.com/withnic/hayate/internal/cli"
	"github.com/withnic/hayate/internal/infra/cache"
	"github.com/withnic/hayate/internal/infra/config"
	"github.com/withnic/hayate/internal/infra/osenv"
	"github.com/withnic/hayate/internal/infra/scanner"
	"github.com/withnic/hayate/internal/infra/tui"
)

var Version = "dev"

func main() {
	home, err := os.UserHomeDir()
	cachePath := ""
	if err == nil {
		cachePath = filepath.Join(home, ".hayate_cache.json")
	}
	service := app.SelectUseCase{
		Load: config.Load, Scanner: scanner.Scanner{}, Cache: cache.FileCache{Path: cachePath}, UI: tui.FuzzySelector{},
		ForShell: osenv.ForShell, Out: os.Stdout, Err: os.Stderr,
	}
	runner := cli.Runner{UseCase: service, InitConfig: config.Init, DefaultShell: osenv.Shell, Version: Version, Out: os.Stdout, Err: os.Stderr}
	os.Exit(runner.Run(context.Background(), os.Args[1:]))
}
