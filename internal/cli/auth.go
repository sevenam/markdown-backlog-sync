package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sevenam/markdown-backlog-sync/internal/auth"
	"github.com/sevenam/markdown-backlog-sync/internal/auth/keyring"
	"github.com/sevenam/markdown-backlog-sync/internal/cli/exitcode"
	"github.com/sevenam/markdown-backlog-sync/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage stored credentials (Personal Access Tokens)",
	}
	cmd.AddCommand(newAuthLoginCmd(g), newAuthLogoutCmd(g), newAuthStatusCmd(g))
	return cmd
}

func newAuthLoginCmd(g *GlobalFlags) *cobra.Command {
	var providerName, tokenFlag string
	var useFile bool
	var fileFlag string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store a PAT for the named provider",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if providerName == "" {
				return exitcode.Errorf(exitcode.Usage, "--provider is required")
			}
			_, cfg, err := loadWorkspaceAndConfig(g)
			if err != nil {
				return err
			}
			if _, ok := cfg.Provider(providerName); !ok {
				return exitcode.Errorf(exitcode.Usage, "no provider named %q in config", providerName)
			}
			be, err := chooseBackend(useFile, fileFlag, g.WorkspaceDir)
			if err != nil {
				return err
			}
			r := newResolverFromConfig(be, cfg)
			tok := tokenFlag
			if tok == "" {
				tok, err = readTokenInteractively(cmd)
				if err != nil {
					return exitcode.Wrap(exitcode.Usage, err)
				}
			}
			if err := r.Login(providerName, tok); err != nil {
				return exitcode.Wrap(exitcode.Auth, err)
			}
			fmt.Fprintf(out(cmd, g.Quiet), "stored credential for %q in %s\n", providerName, be.Name())
			return nil
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "provider name (must match a config entry)")
	cmd.Flags().StringVar(&tokenFlag, "token", "", "token value (discouraged: prefer interactive prompt)")
	cmd.Flags().BoolVar(&useFile, "file", false, "force use of the encrypted file backend")
	cmd.Flags().StringVar(&fileFlag, "credentials-file", "", "path to encrypted credentials file (implies --file)")
	return cmd
}

func newAuthLogoutCmd(g *GlobalFlags) *cobra.Command {
	var providerName string
	var useFile bool
	var fileFlag string
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored PAT for the named provider",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if providerName == "" {
				return exitcode.Errorf(exitcode.Usage, "--provider is required")
			}
			_, cfg, err := loadWorkspaceAndConfig(g)
			if err != nil {
				return err
			}
			be, err := chooseBackend(useFile, fileFlag, g.WorkspaceDir)
			if err != nil {
				return err
			}
			r := newResolverFromConfig(be, cfg)
			if err := r.Logout(providerName); err != nil {
				if errors.Is(err, auth.ErrNotFound) {
					fmt.Fprintf(out(cmd, g.Quiet), "no stored credential for %q\n", providerName)
					return nil
				}
				return exitcode.Wrap(exitcode.Auth, err)
			}
			fmt.Fprintf(out(cmd, g.Quiet), "removed credential for %q\n", providerName)
			return nil
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "provider name")
	cmd.Flags().BoolVar(&useFile, "file", false, "force use of the encrypted file backend")
	cmd.Flags().StringVar(&fileFlag, "credentials-file", "", "path to encrypted credentials file")
	return cmd
}

// newResolverFromConfig builds a Resolver and pre-populates ProviderKinds
// and ProviderScopes from the given config so backend account keys are
// scoped to the actual remote, not just the local provider name.
func newResolverFromConfig(be auth.Backend, cfg *config.Config) *auth.Resolver {
	r := auth.NewResolver(be)
	for _, p := range cfg.Providers {
		r.ProviderKinds[p.Name] = p.Type
		if scope := scopeFromOptions(p.Type, p.Options); scope != "" {
			r.ProviderScopes[p.Name] = scope
		}
	}
	return r
}

// scopeFromOptions extracts a stable remote-identity string for known
// provider kinds. Returning "" disables scope-based collision protection.
func scopeFromOptions(kind string, opts map[string]any) string {
	str := func(k string) string {
		if v, ok := opts[k].(string); ok {
			return v
		}
		return ""
	}
	switch kind {
	case "github-issues", "github-projects-v2":
		return str("repo") // owner/repo or owner/projectNumber
	case "azure-devops-boards":
		org, proj := str("organization"), str("project")
		if org == "" && proj == "" {
			return ""
		}
		return org + "/" + proj
	}
	return ""
}

func newAuthStatusCmd(g *GlobalFlags) *cobra.Command {
	var useFile bool
	var fileFlag string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show which providers have stored credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := loadWorkspaceAndConfig(g)
			if err != nil {
				return err
			}
			be, err := chooseBackend(useFile, fileFlag, g.WorkspaceDir)
			if err != nil {
				return err
			}
			r := newResolverFromConfig(be, cfg)
			w := out(cmd, g.Quiet)
			fmt.Fprintf(w, "credential backend: %s\n", be.Name())
			names := make([]string, 0, len(cfg.Providers))
			for _, p := range cfg.Providers {
				names = append(names, p.Name)
			}
			sort.Strings(names)
			for _, n := range names {
				has, err := r.Has(n)
				switch {
				case err != nil:
					fmt.Fprintf(w, "  %s: error (%v)\n", n, err)
				case has:
					fmt.Fprintf(w, "  %s: stored\n", n)
				default:
					fmt.Fprintf(w, "  %s: missing\n", n)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&useFile, "file", false, "force use of the encrypted file backend")
	cmd.Flags().StringVar(&fileFlag, "credentials-file", "", "path to encrypted credentials file")
	return cmd
}

// chooseBackend selects the appropriate credential backend.
func chooseBackend(useFile bool, fileFlag, workspaceDir string) (auth.Backend, error) {
	if useFile || fileFlag != "" {
		path := fileFlag
		if path == "" {
			path = defaultCredentialsFile(workspaceDir)
		}
		passphrase := envOrDefault("MBS_CREDENTIALS_PASSPHRASE", "")
		if passphrase == "" {
			return nil, exitcode.Errorf(exitcode.Auth, "MBS_CREDENTIALS_PASSPHRASE must be set when using the file backend")
		}
		return auth.NewFileBackend(path, passphrase), nil
	}
	if keyring.Available(auth.Service) {
		return keyring.New(auth.Service), nil
	}
	return nil, exitcode.Errorf(exitcode.Auth, "no system keyring available; pass --file with MBS_CREDENTIALS_PASSPHRASE set")
}

func defaultCredentialsFile(workspaceDir string) string {
	// Keep credentials out of the workspace by default — .sync/ may be
	// committed for team-wide sync conflict detection, and credentials
	// must never end up in version control.
	if home, err := os.UserConfigDir(); err == nil {
		return filepath.Join(home, "markdown-backlog-sync", "credentials.bin")
	}
	if workspaceDir != "" {
		return filepath.Join(workspaceDir, ".sync", "credentials.bin")
	}
	return "./mbs-credentials.bin"
}

// readTokenInteractively prompts the user for a token, masking input when
// stdin is a terminal.
func readTokenInteractively(cmd *cobra.Command) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), "Enter token: ")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
