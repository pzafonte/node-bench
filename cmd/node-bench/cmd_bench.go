package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pzafonte/node-bench/internal/runner"
	"github.com/spf13/cobra"
)

func cmdBench() *cobra.Command {
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
		Use:   "bench <node-repo> <ref-a> <ref-b>",
		Short: "Benchmark two git refs and print a side-by-side comparison",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath, refA, refB := args[0], args[1], args[2]

			prof, err := resolveProfile(profileName, repoPath)
			if err != nil {
				return err
			}

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

			return runBench(cmd.OutOrStdout(), repoPath, refA, refB, cfg, resultsDir)
		},
	}

	cmd.Flags().IntVar(&duration, "duration", 300, "seconds to run the node per trial")
	cmd.Flags().StringVar(&network, "network", "signet", "network (signet, mainnet, testnet)")
	cmd.Flags().StringVar(&connect, "connect", "",
		"peer(s) to connect to, comma-separated (each trial picks one at random; default: profile default_peers)")
	cmd.Flags().IntVar(&trials, "trials", 1, "number of independent trials per ref")
	cmd.Flags().StringVar(&profileName, "profile", "", "node profile name, directory, or .toml path (default: auto-detect from repo)")
	cmd.Flags().StringVar(&resultsDir, "results-dir", "results", "directory to write result JSON files")
	cmd.Flags().StringVar(&logsDir, "logs-dir", "logs", "directory to write log files")
	return cmd
}

func runBench(w io.Writer, repoPath, refA, refB string, cfg runner.Config, resultsDir string) error {
	origRef, err := gitHeadRef(repoPath)
	if err != nil {
		return fmt.Errorf("get current ref: %w", err)
	}

	// Restore the original branch when done, whether we succeed or fail.
	restored := false
	defer func() {
		if !restored {
			_ = gitCheckout(repoPath, origRef)
		}
	}()

	fmt.Fprintf(w, "\n=== Benchmarking %s ===\n", refA)
	if err := gitCheckout(repoPath, refA); err != nil {
		return err
	}
	rA, err := runner.Run(cfg)
	if err != nil {
		return fmt.Errorf("bench %s: %w", refA, err)
	}
	pathA, err := rA.Save(resultsDir)
	if err != nil {
		return fmt.Errorf("save result for %s: %w", refA, err)
	}
	fmt.Fprintf(w, "  saved: %s\n", pathA)

	fmt.Fprintf(w, "\n=== Benchmarking %s ===\n", refB)
	if err := gitCheckout(repoPath, refB); err != nil {
		return err
	}
	rB, err := runner.Run(cfg)
	if err != nil {
		return fmt.Errorf("bench %s: %w", refB, err)
	}
	pathB, err := rB.Save(resultsDir)
	if err != nil {
		return fmt.Errorf("save result for %s: %w", refB, err)
	}
	fmt.Fprintf(w, "  saved: %s\n", pathB)

	if err := gitCheckout(repoPath, origRef); err != nil {
		return err
	}
	restored = true

	printComparison(w, rA.Commit, rB.Commit, rA, rB)
	return nil
}

// gitHeadRef returns the current branch name, or the short SHA when HEAD is detached.
func gitHeadRef(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	ref := strings.TrimSpace(string(out))
	if ref != "HEAD" {
		return ref, nil
	}
	// Detached HEAD: record the short SHA so checkout restores the same commit.
	out2, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --short: %w", err)
	}
	return strings.TrimSpace(string(out2)), nil
}

// gitCheckout checks out ref in the repository at dir.
// -f discards tracked-file modifications left by the build toolchain (e.g.
// Cargo.lock changes from cargo build). Untracked files are not affected.
func gitCheckout(dir, ref string) error {
	cmd := exec.Command("git", "-C", dir, "checkout", "-f", ref)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout %s: %w", ref, err)
	}
	return nil
}
