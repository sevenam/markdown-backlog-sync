package cli

import (
	"fmt"
	"sort"

	"github.com/sevenam/markdown-backlog-sync/internal/cli/exitcode"
	"github.com/spf13/cobra"
)

func newProviderCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Inspect configured providers",
	}
	cmd.AddCommand(newProviderListCmd(g))
	return cmd
}

func newProviderListCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List providers configured in backlog.config.toml",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := loadWorkspaceAndConfig(g)
			if err != nil {
				return err
			}
			if len(cfg.Providers) == 0 {
				fmt.Fprintln(out(cmd, g.Quiet), "no providers configured")
				return nil
			}
			lines := make([]string, 0, len(cfg.Providers))
			for _, p := range cfg.Providers {
				lines = append(lines, fmt.Sprintf("%-20s %s", p.Name, p.Type))
			}
			sort.Strings(lines)
			for _, l := range lines {
				fmt.Fprintln(out(cmd, g.Quiet), l)
			}
			return nil
		},
	}
}

// Compile-time guard that exitcode is used (keeps the import even if the
// file is later edited to drop direct uses).
var _ = exitcode.OK
