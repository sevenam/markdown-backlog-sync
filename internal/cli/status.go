package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sevenam/markdown-backlog-sync/internal/cli/exitcode"
	"github.com/sevenam/markdown-backlog-sync/internal/config"
	"github.com/sevenam/markdown-backlog-sync/internal/markdown"
	"github.com/sevenam/markdown-backlog-sync/internal/workspace"
	"github.com/spf13/cobra"
)

func newStatusCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show workspace and provider status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, cfg, err := loadWorkspaceAndConfig(g)
			if err != nil {
				return err
			}
			w := out(cmd, g.Quiet)
			fmt.Fprintf(w, "workspace: %s\n", ws.Root)
			fmt.Fprintf(w, "items dir: %s\n", ws.ItemsDir)
			fmt.Fprintf(w, "state dir: %s\n", ws.StateDir)
			fmt.Fprintf(w, "providers (%d):\n", len(cfg.Providers))
			names := make([]string, 0, len(cfg.Providers))
			for _, p := range cfg.Providers {
				names = append(names, fmt.Sprintf("  - %s (%s)", p.Name, p.Type))
			}
			sort.Strings(names)
			for _, l := range names {
				fmt.Fprintln(w, l)
			}
			items, err := listItemFiles(ws.ItemsDir)
			if err != nil {
				return exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
			}
			fmt.Fprintf(w, "items: %d markdown file(s) under %s\n", len(items), ws.ItemsDir)
			return nil
		},
	}
}

func newFmtCmd(g *GlobalFlags) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "fmt [paths...]",
		Short: "Rewrite item files into canonical Markdown form",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, _, err := loadWorkspaceAndConfig(g)
			if err != nil {
				return err
			}
			paths := args
			if len(paths) == 0 {
				paths, err = listItemFiles(ws.ItemsDir)
				if err != nil {
					return exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
				}
			}
			var changed []string
			for _, p := range paths {
				b, err := os.ReadFile(p)
				if err != nil {
					return exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
				}
				it, err := markdown.Parse(string(b))
				if err != nil {
					return exitcode.Errorf(exitcode.WorkspaceIntegrity, "%s: %v", p, err)
				}
				out, err := markdown.Serialize(it)
				if err != nil {
					return exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
				}
				if out == string(b) {
					continue
				}
				changed = append(changed, p)
				if check || g.DryRun {
					continue
				}
				if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
					return exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
				}
			}
			w := out(cmd, g.Quiet)
			for _, p := range changed {
				fmt.Fprintln(w, p)
			}
			if check && len(changed) > 0 {
				return exitcode.Errorf(exitcode.WorkspaceIntegrity, "%d file(s) need formatting", len(changed))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "exit non-zero if any files would be rewritten; do not modify them")
	return cmd
}

// loadWorkspaceAndConfig is shared by commands that need both.
func loadWorkspaceAndConfig(g *GlobalFlags) (*workspace.Workspace, *config.Config, error) {
	ws, err := workspace.Open(g.WorkspaceDir, "")
	if err != nil {
		return nil, nil, exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
	}
	cfgPath := g.ConfigFile
	if cfgPath == "" {
		cfgPath = ws.ConfigPath
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, nil, exitcode.Wrap(exitcode.Usage, err)
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, nil, exitcode.Wrap(exitcode.Usage, err)
	}
	// Re-resolve workspace with configured items dir so callers see the
	// final paths.
	ws, err = workspace.Open(g.WorkspaceDir, cfg.Workspace.ItemsDir)
	if err != nil {
		return nil, nil, exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
	}
	return ws, cfg, nil
}

// listItemFiles returns absolute paths to *.md files under root, recursively.
func listItemFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == root {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
