// Command mbs is the markdown-backlog-sync CLI entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sevenam/markdown-backlog-sync/internal/cli"
	"github.com/sevenam/markdown-backlog-sync/internal/cli/exitcode"
)

// Build metadata, populated via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := cli.NewRootCmd(cli.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})

	if err := root.ExecuteContext(ctx); err != nil {
		var ce *exitcode.Error
		if errors.As(err, &ce) {
			fmt.Fprintln(os.Stderr, "error:", ce.Error())
			os.Exit(int(ce.Code))
		}
		fmt.Fprintln(os.Stderr, "error:", err.Error())
		os.Exit(int(exitcode.Generic))
	}
}
