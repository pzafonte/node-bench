package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pzafonte/node-bench/internal/nodeprofile"
	"github.com/pzafonte/node-bench/internal/runner"
	"github.com/spf13/cobra"
)

func cmdRun() *cobra.Command {
	var (
		duration    int
		network     string
		connect     string
		trials      int
		profileName string
		resultsDir  string
		logsDir     string
	)

	cmd := &cobra.Command{
		Use:   "run <node-repo>",
		Short: "Build and run a node, storing results as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath := args[0]
			prof, err := resolveProfile(profileName, repoPath)
			if err != nil {
				return err
			}

			// Peer resolution: --connect flag > profile default_peers > error.
			var peers []string
			if connect != "" {
				peers = strings.Split(connect, ",")
			} else if len(prof.DefaultPeers) > 0 {
				peers = prof.DefaultPeers
			}
			if len(peers) == 0 {
				return fmt.Errorf(
					"no peers configured: pass --connect <addr,...> or add default_peers to the profile\n" +
						"  example: --connect 45.94.168.5:38333",
				)
			}

			cfg := runner.Config{
				RepoPath: repoPath,
				Profile:  prof,
				Network:  network,
				Duration: time.Duration(duration) * time.Second,
				Trials:   trials,
				Peers:    peers,
				LogsDir:  logsDir,
			}

			r, err := runner.Run(cfg)
			if err != nil {
				return err
			}

			path, err := r.Save(resultsDir)
			if err != nil {
				return fmt.Errorf("save result: %w", err)
			}

			fmt.Printf("Result saved to %s\n", path)
			fmt.Printf("  commit=%s  branch=%s  trials=%d\n", r.Commit, r.Branch, r.Trials)
			fmt.Printf("  header_sync_s=%d  block_sync_s=%d  bps=%.1f  height=%d\n",
				r.HeaderSyncS, r.BlockSyncS, r.BlocksPerSec, r.MaxHeight)
			return nil
		},
	}

	cmd.Flags().IntVar(&duration, "duration", 300, "seconds to run the node per trial")
	cmd.Flags().StringVar(&network, "network", "signet", "network (signet, mainnet, testnet)")
	cmd.Flags().StringVar(&connect, "connect", "",
		"peer(s) to connect to, comma-separated (each trial picks one at random; default: profile default_peers)")
	cmd.Flags().IntVar(&trials, "trials", 1, "number of independent trials to run")
	cmd.Flags().StringVar(&profileName, "profile", "", "node profile name, directory, or .toml path (default: auto-detect from repo)")
	cmd.Flags().StringVar(&resultsDir, "results-dir", "results", "directory to write result JSON files")
	cmd.Flags().StringVar(&logsDir, "logs-dir", "logs", "directory to write log files")
	return cmd
}

// fileExists reports whether path is an existing regular file.
func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

// resolveProfile picks the NodeProfile to use for a run.
// Priority: explicit --profile flag > .node-bench.toml auto-detected in repo dir.
// Returns an error if neither source provides a profile.
func resolveProfile(flag, repoPath string) (*nodeprofile.NodeProfile, error) {
	if flag != "" {
		return nodeprofile.Resolve(flag)
	}
	// Try the repo directory (auto-detects .node-bench.toml inside it).
	if p, err := nodeprofile.Resolve(repoPath); err == nil {
		return p, nil
	}
	return nil, fmt.Errorf(
		"no profile found: add a .node-bench.toml to %s or pass --profile <path>"+
			"\n  reference profiles: profiles/kernel-node.toml  profiles/btcd.toml",
		repoPath,
	)
}
