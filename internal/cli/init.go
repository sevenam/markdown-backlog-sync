package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sevenam/markdown-backlog-sync/internal/cli/exitcode"
	"github.com/sevenam/markdown-backlog-sync/internal/workspace"
	"github.com/spf13/cobra"
)

func newInitCmd(g *GlobalFlags) *cobra.Command {
	var itemsDir string
	var overwrite bool
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new workspace in the current or given directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := g.WorkspaceDir
			if len(args) == 1 {
				root = args[0]
			}
			if root == "" {
				wd, err := os.Getwd()
				if err != nil {
					return exitcode.Wrap(exitcode.Generic, err)
				}
				root = wd
			}
			ws, wrote, err := workspace.Create(root, itemsDir, overwrite)
			if err != nil {
				return exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
			}
			w := out(cmd, g.Quiet)
			if wrote {
				fmt.Fprintf(w, "initialized workspace at %s\n", ws.Root)
			} else {
				fmt.Fprintf(w, "workspace already initialized at %s\n", ws.Root)
			}
			fmt.Fprintf(w, "  config:    %s\n", ws.ConfigPath)
			fmt.Fprintf(w, "  items dir: %s\n", ws.ItemsDir)
			fmt.Fprintf(w, "  state dir: %s\n", ws.StateDir)
			// Best-effort: append .sync/ to .gitignore if a git repo is detected.
			if err := maybeUpdateGitignore(ws.Root); err != nil && g.Verbose {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&itemsDir, "items-dir", "", "relative path to the items directory (default: backlog)")
	cmd.Flags().BoolVar(&overwrite, "force", false, "overwrite an existing config file")
	return cmd
}

func maybeUpdateGitignore(root string) error {
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return nil // not a git repo
	}
	gi := filepath.Join(root, ".gitignore")
	existing, err := os.ReadFile(gi)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if containsLine(string(existing), workspace.StateDirName+"/") {
		return nil
	}
	f, err := os.OpenFile(gi, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		_, _ = f.WriteString("\n")
	}
	_, err = fmt.Fprintf(f, "%s/\n", workspace.StateDirName)
	return err
}

func containsLine(haystack, needle string) bool {
	for len(haystack) > 0 {
		var line string
		if i := indexByte(haystack, '\n'); i >= 0 {
			line, haystack = haystack[:i], haystack[i+1:]
		} else {
			line, haystack = haystack, ""
		}
		if line == needle {
			return true
		}
	}
	return false
}

// avoid importing strings just for this
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
