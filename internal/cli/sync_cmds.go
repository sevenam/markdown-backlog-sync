package cli

import (
	"fmt"

	"github.com/sevenam/markdown-backlog-sync/internal/cli/exitcode"
	"github.com/spf13/cobra"
)

// notImplemented is a placeholder used by command stubs that require
// subsequent phases (sync engine + concrete providers) to do real work.
func notImplemented(cmd *cobra.Command, what string) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "%s is not yet implemented in this build (Phase 1 foundation only)\n", what)
	return exitcode.Errorf(exitcode.Usage, "%s requires a provider implementation", what)
}

func newPullCmd(g *GlobalFlags) *cobra.Command {
	var providerName string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull remote items into local Markdown files",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented(cmd, "pull")
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "scope to a single provider")
	_ = g
	return cmd
}

func newPushCmd(g *GlobalFlags) *cobra.Command {
	var providerName string
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push local Markdown files to the remote",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented(cmd, "push")
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "scope to a single provider")
	_ = g
	return cmd
}

func newSyncCmd(g *GlobalFlags) *cobra.Command {
	var providerName string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Bidirectionally synchronize local Markdown files with the remote",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented(cmd, "sync")
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "scope to a single provider")
	_ = g
	return cmd
}
