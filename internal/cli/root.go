// Package cli wires the cobra command tree for the mbs CLI.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// BuildInfo carries release metadata injected at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// GlobalFlags holds values bound to root persistent flags.
type GlobalFlags struct {
	WorkspaceDir string
	ConfigFile   string
	Verbose      bool
	Quiet        bool
	JSON         bool
	DryRun       bool
}

// NewRootCmd builds the root cobra command and attaches all subcommands.
func NewRootCmd(info BuildInfo) *cobra.Command {
	flags := &GlobalFlags{}

	root := &cobra.Command{
		Use:           "mbs",
		Short:         "Sync local Markdown files with Azure DevOps Boards and GitHub Issues",
		Long:          "mbs (markdown-backlog-sync) treats local Markdown files as the source of truth for a software backlog and syncs them with remote backlogs.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", info.Version, info.Commit, info.Date),
	}

	pf := root.PersistentFlags()
	pf.StringVarP(&flags.WorkspaceDir, "workspace", "w", "", "path to workspace directory (default: discovered from cwd)")
	pf.StringVar(&flags.ConfigFile, "config", "", "path to config file (default: <workspace>/backlog.config.toml)")
	pf.BoolVarP(&flags.Verbose, "verbose", "v", false, "verbose logging")
	pf.BoolVarP(&flags.Quiet, "quiet", "q", false, "suppress non-error output")
	pf.BoolVar(&flags.JSON, "json", false, "emit machine-readable JSON output")
	pf.BoolVar(&flags.DryRun, "dry-run", false, "describe actions without making changes")

	root.AddCommand(
		newInitCmd(flags),
		newStatusCmd(flags),
		newPullCmd(flags),
		newPushCmd(flags),
		newSyncCmd(flags),
		newFmtCmd(flags),
		newAuthCmd(flags),
		newProviderCmd(flags),
		newVersionCmd(info),
	)
	return root
}

// out returns the writer to print to (stdout unless quiet).
func out(cmd *cobra.Command, quiet bool) io.Writer {
	if quiet {
		return io.Discard
	}
	return cmd.OutOrStdout()
}

// envOrDefault returns the env var value if set, otherwise def.
func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
