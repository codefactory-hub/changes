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
	_ = version
	_ = commit
	_ = date

	app := cli.NewApp(os.Stdout, os.Stderr)
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
