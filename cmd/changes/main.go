package main

import (
	"context"
	"os"

	"github.com/example/changes/internal/cli"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	app := cli.NewApp(os.Stdout, os.Stderr)
	app.Version = version
	app.Commit = commit
	app.Date = date
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
