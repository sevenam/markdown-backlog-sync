package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sevenam/markdown-backlog-sync/internal/cli/exitcode"
	"github.com/sevenam/markdown-backlog-sync/internal/config"
	"github.com/sevenam/markdown-backlog-sync/internal/markdown"
	"github.com/sevenam/markdown-backlog-sync/internal/provider"
	"github.com/sevenam/markdown-backlog-sync/internal/state"
	"github.com/sevenam/markdown-backlog-sync/internal/workspace"
	"github.com/spf13/cobra"
)

// notImplemented is a placeholder used by command stubs that require
// subsequent phases (sync engine + concrete providers) to do real work.
func notImplemented(cmd *cobra.Command, what string) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "%s is not yet implemented in this build\n", what)
	return exitcode.Errorf(exitcode.Usage, "%s requires a full sync engine implementation", what)
}

func newPullCmd(g *GlobalFlags) *cobra.Command {
	var providerName string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull remote items into local Markdown files",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, cfg, err := loadWorkspaceAndConfig(g)
			if err != nil {
				return err
			}

			providers := cfg.Providers
			if providerName != "" {
				pCfg, ok := cfg.Provider(providerName)
				if !ok {
					return exitcode.Errorf(exitcode.Usage, "no provider named %q in config", providerName)
				}
				providers = []config.ProviderConfig{pCfg}
			}
			if len(providers) == 0 {
				fmt.Fprintln(out(cmd, g.Quiet), "no providers configured")
				return nil
			}

			be, err := chooseBackend(false, "", g.WorkspaceDir)
			if err != nil {
				return err
			}
			resolver := newResolverFromConfig(be, cfg)
			reg := newProviderRegistry()

			store, err := state.Open(ws.StateDir)
			if err != nil {
				return exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
			}

			w := out(cmd, g.Quiet)
			for _, pCfg := range providers {
				prov, err := reg.Build(pCfg.Type, pCfg.Name, pCfg.Options, resolver)
				if err != nil {
					if errors.Is(err, provider.ErrAuth) {
						return exitcode.Wrap(exitcode.Auth, fmt.Errorf("provider %q: %w", pCfg.Name, err))
					}
					if errors.Is(err, provider.ErrUnknownType) {
						return exitcode.Errorf(exitcode.Usage, "provider %q has unknown type %q", pCfg.Name, pCfg.Type)
					}
					return exitcode.Wrap(exitcode.Generic, fmt.Errorf("provider %q: %w", pCfg.Name, err))
				}

				fmt.Fprintf(w, "pulling from %s (%s)...\n", pCfg.Name, pCfg.Type)
				n, err := pullProvider(cmd.Context(), prov, ws, store, g.DryRun, g.Verbose, w)
				if err != nil {
					if errors.Is(err, provider.ErrAuth) {
						return exitcode.Wrap(exitcode.Auth, fmt.Errorf("provider %q: %w", pCfg.Name, err))
					}
					if errors.Is(err, provider.ErrRateLimited) {
						return exitcode.Wrap(exitcode.Generic, fmt.Errorf("provider %q: rate limited — wait and retry", pCfg.Name))
					}
					return exitcode.Wrap(exitcode.Generic, fmt.Errorf("provider %q: %w", pCfg.Name, err))
				}
				if g.DryRun {
					fmt.Fprintf(w, "  [dry-run] %d item(s) would be pulled from %s\n", n, pCfg.Name)
				} else {
					fmt.Fprintf(w, "  %d item(s) pulled from %s\n", n, pCfg.Name)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "scope to a single provider")
	return cmd
}

// pullProvider fetches all items from prov and writes them into ws.ItemsDir,
// updating the state store. Returns the count of items processed.
func pullProvider(
	ctx context.Context,
	prov provider.Provider,
	ws *workspace.Workspace,
	store *state.Store,
	dryRun, verbose bool,
	w io.Writer,
) (int, error) {
	items, err := prov.List(ctx, nil)
	if err != nil {
		return 0, err
	}

	idx, err := store.LoadIndex()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, item := range items {
		mapKey := state.MapKey(prov.Name(), item.ID)
		existingLocalID, exists := idx.Mapping[mapKey]

		mdItem := remoteItemToMarkdown(item)
		serialized, err := markdown.Serialize(mdItem)
		if err != nil {
			return count, fmt.Errorf("serialize item %s: %w", item.ID, err)
		}

		var localID, localPath string
		if exists {
			ist, err := store.LoadItem(existingLocalID)
			if err == nil && ist != nil {
				localID = existingLocalID
				localPath = ist.LocalPath
			} else {
				// State entry missing — treat as new.
				exists = false
			}
		}
		if !exists {
			localID = safeLocalID(prov.Name(), item.ID)
			localPath = filepath.Join(ws.ItemsDir, safeFilename(item.ID, item.Title))
		}

		// Rename existing files to the canonical issue-number-prefixed name,
		// e.g. "core-cli-skeleton.md" → "7-core-cli-skeleton-cobra-viper.md".
		// This happens on pull so every tracked file carries its remote ID.
		canonicalPath := filepath.Join(ws.ItemsDir, safeFilename(item.ID, item.Title))
		needsRename := exists && filepath.Clean(localPath) != filepath.Clean(canonicalPath)
		if needsRename {
			if verbose {
				fmt.Fprintf(w, "  [rename] %s → %s\n", filepath.Base(localPath), filepath.Base(canonicalPath))
			}
			if !dryRun {
				if renameErr := os.Rename(localPath, canonicalPath); renameErr != nil {
					if !errors.Is(renameErr, os.ErrNotExist) {
						return count, fmt.Errorf("rename %s: %w", filepath.Base(localPath), renameErr)
					}
					// Old file already gone — just write to canonical path.
				}
				localPath = canonicalPath
			}
		}

		if verbose {
			action := "update"
			if !exists {
				action = "create"
			}
			fmt.Fprintf(w, "  [%s] #%s %s\n", action, item.ID, item.Title)
		}

		if !dryRun {
			if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
				return count, err
			}
			if err := os.WriteFile(localPath, []byte(serialized), 0o644); err != nil {
				return count, err
			}

			ist := &state.ItemState{
				LocalID:      localID,
				LocalPath:    localPath,
				Provider:     prov.Name(),
				RemoteID:     item.ID,
				RemoteURL:    item.URL,
				RemoteRev:    item.Rev,
				ContentHash:  state.Hash(serialized),
				LastSyncedAt: time.Now().UTC().Format(time.RFC3339),
			}
			if err := store.SaveItem(ist); err != nil {
				return count, err
			}
			if err := store.SaveSnapshot(localID, serialized); err != nil {
				return count, err
			}
			idx.Mapping[mapKey] = localID
		}
		count++
	}

	if !dryRun {
		if err := store.SaveIndex(idx); err != nil {
			return count, err
		}
	}
	return count, nil
}

func newPushCmd(g *GlobalFlags) *cobra.Command {
	var providerName string
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push local Markdown files to the remote",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, cfg, err := loadWorkspaceAndConfig(g)
			if err != nil {
				return err
			}

			providers := cfg.Providers
			if providerName != "" {
				pCfg, ok := cfg.Provider(providerName)
				if !ok {
					return exitcode.Errorf(exitcode.Usage, "no provider named %q in config", providerName)
				}
				providers = []config.ProviderConfig{pCfg}
			}
			if len(providers) == 0 {
				fmt.Fprintln(out(cmd, g.Quiet), "no providers configured")
				return nil
			}

			// Only create new (untracked) items when the target provider is unambiguous.
			createNew := len(providers) == 1 || providerName != ""

			be, err := chooseBackend(false, "", g.WorkspaceDir)
			if err != nil {
				return err
			}
			resolver := newResolverFromConfig(be, cfg)
			reg := newProviderRegistry()

			store, err := state.Open(ws.StateDir)
			if err != nil {
				return exitcode.Wrap(exitcode.WorkspaceIntegrity, err)
			}

			w := out(cmd, g.Quiet)
			for _, pCfg := range providers {
				prov, err := reg.Build(pCfg.Type, pCfg.Name, pCfg.Options, resolver)
				if err != nil {
					if errors.Is(err, provider.ErrAuth) {
						return exitcode.Wrap(exitcode.Auth, fmt.Errorf("provider %q: %w", pCfg.Name, err))
					}
					if errors.Is(err, provider.ErrUnknownType) {
						return exitcode.Errorf(exitcode.Usage, "provider %q has unknown type %q", pCfg.Name, pCfg.Type)
					}
					return exitcode.Wrap(exitcode.Generic, fmt.Errorf("provider %q: %w", pCfg.Name, err))
				}

				fmt.Fprintf(w, "pushing to %s (%s)...\n", pCfg.Name, pCfg.Type)
				nCreated, nUpdated, err := pushProvider(cmd.Context(), prov, ws, store, g.DryRun, g.Verbose, createNew, w)
				if err != nil {
					if errors.Is(err, provider.ErrAuth) {
						return exitcode.Wrap(exitcode.Auth, fmt.Errorf("provider %q: %w", pCfg.Name, err))
					}
					if errors.Is(err, provider.ErrRateLimited) {
						return exitcode.Wrap(exitcode.Generic, fmt.Errorf("provider %q: rate limited — wait and retry", pCfg.Name))
					}
					if errors.Is(err, provider.ErrConflict) {
						return exitcode.Wrap(exitcode.Generic, fmt.Errorf("provider %q: conflict detected — run pull first", pCfg.Name))
					}
					return exitcode.Wrap(exitcode.Generic, fmt.Errorf("provider %q: %w", pCfg.Name, err))
				}
				prefix := ""
				if g.DryRun {
					prefix = "[dry-run] "
				}
				fmt.Fprintf(w, "  %s%d created, %d updated\n", prefix, nCreated, nUpdated)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "scope to a single provider")
	return cmd
}

// pushProvider reads all Markdown files in ws.ItemsDir and pushes local
// changes to the remote provider.
//   - Items with a tracked RemoteId: pushed as updates if the content hash
//     has changed since the last sync.
//   - Items without a RemoteId: created on the provider when createNew is true.
//
// Returns (created, updated, error).
func pushProvider(
	ctx context.Context,
	prov provider.Provider,
	ws *workspace.Workspace,
	store *state.Store,
	dryRun, verbose, createNew bool,
	w io.Writer,
) (int, int, error) {
	idx, err := store.LoadIndex()
	if err != nil {
		return 0, 0, err
	}

	entries, err := os.ReadDir(ws.ItemsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	var created, updated int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		localPath := filepath.Join(ws.ItemsDir, entry.Name())
		data, err := os.ReadFile(localPath)
		if err != nil {
			return created, updated, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		content := string(data)

		mdItem, err := markdown.Parse(content)
		if err != nil {
			return created, updated, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		remoteID, hasRemoteID := mdItem.Properties.Get("RemoteId")

		if hasRemoteID && remoteID != "" {
			// Tracked item — check if it belongs to this provider.
			mapKey := state.MapKey(prov.Name(), remoteID)
			localID, tracked := idx.Mapping[mapKey]
			if !tracked {
				// Not in the index for this provider. If multiple providers are
				// configured, assume the RemoteId belongs to a different one and
				// skip. With a single provider (or --provider), try to adopt it
				// as a force-update (handles wiped .sync/, cross-machine sync, etc.).
				if !createNew {
					continue
				}
				// Adopt: register in index and fall through to the update path.
				localID = safeLocalID(prov.Name(), remoteID)
				idx.Mapping[mapKey] = localID
			}

			ist, err := store.LoadItem(localID)
			if err != nil {
				return created, updated, fmt.Errorf("load state for %s: %w", localID, err)
			}
			if ist == nil {
				ist = &state.ItemState{
					LocalID:   localID,
					LocalPath: localPath,
					Provider:  prov.Name(),
					RemoteID:  remoteID,
				}
			}

			currentHash := state.Hash(content)
			if currentHash == ist.ContentHash {
				if verbose {
					fmt.Fprintf(w, "  [skip] #%s %s (unchanged)\n", remoteID, mdItem.Title)
				}
				continue
			}

			patch := markdownToItemPatch(mdItem)
			if verbose {
				fmt.Fprintf(w, "  [update] #%s %s\n", remoteID, mdItem.Title)
			}
			if !dryRun {
				updatedItem, err := prov.Update(ctx, remoteID, patch, ist.RemoteRev)
				if err != nil {
					return created, updated, fmt.Errorf("update #%s: %w", remoteID, err)
				}
				ist.RemoteRev = updatedItem.Rev
				ist.ContentHash = currentHash
				ist.LastSyncedAt = time.Now().UTC().Format(time.RFC3339)
				if err := store.SaveItem(ist); err != nil {
					return created, updated, err
				}
				if err := store.SaveSnapshot(localID, content); err != nil {
					return created, updated, err
				}
			}
			updated++
		} else {
			// New item — no RemoteId.
			if !createNew {
				if verbose {
					fmt.Fprintf(w, "  [skip] %s (no RemoteId; use --provider to assign a target)\n", mdItem.Title)
				}
				continue
			}
			ri := markdownToRemoteItem(mdItem)
			if verbose {
				fmt.Fprintf(w, "  [create] %s\n", mdItem.Title)
			}
			if !dryRun {
				createdItem, err := prov.Create(ctx, ri)
				if err != nil {
					return created, updated, fmt.Errorf("create %q: %w", mdItem.Title, err)
				}
				// Write RemoteId and Url back into the local file, preserving all
				// other content (body, custom sections, etc.).
				mdItem.Properties.Set("RemoteId", createdItem.ID)
				mdItem.Properties.Set("Url", createdItem.URL)
				serialized, err := markdown.Serialize(mdItem)
				if err != nil {
					return created, updated, fmt.Errorf("serialize %q: %w", mdItem.Title, err)
				}
				if err := os.WriteFile(localPath, []byte(serialized), 0o644); err != nil {
					return created, updated, err
				}
				localID := safeLocalID(prov.Name(), createdItem.ID)
				ist := &state.ItemState{
					LocalID:      localID,
					LocalPath:    localPath,
					Provider:     prov.Name(),
					RemoteID:     createdItem.ID,
					RemoteURL:    createdItem.URL,
					RemoteRev:    createdItem.Rev,
					ContentHash:  state.Hash(serialized),
					LastSyncedAt: time.Now().UTC().Format(time.RFC3339),
				}
				if err := store.SaveItem(ist); err != nil {
					return created, updated, err
				}
				if err := store.SaveSnapshot(localID, serialized); err != nil {
					return created, updated, err
				}
				idx.Mapping[state.MapKey(prov.Name(), createdItem.ID)] = localID
			}
			created++
		}
	}

	if !dryRun {
		if err := store.SaveIndex(idx); err != nil {
			return created, updated, err
		}
	}
	return created, updated, nil
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
