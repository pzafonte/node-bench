package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pzafonte/node-bench/internal/logparse"
	"github.com/pzafonte/node-bench/internal/nodeprofile"
	"github.com/pzafonte/node-bench/internal/result"
	"github.com/spf13/cobra"
)

func cmdAnalyze() *cobra.Command {
	var (
		resultsDir  string
		profileName string
	)
	cmd := &cobra.Command{
		Use:   "analyze <log-file> <commit> <branch> <network> <duration-s>",
		Short: "Parse an existing log file and write a JSON result",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(args, resultsDir, profileName)
		},
	}
	cmd.Flags().StringVar(&resultsDir, "results-dir", "results", "directory to write result JSON files")
	cmd.Flags().StringVar(&profileName, "profile", "", "path to .node-bench.toml or directory containing one (required)")
	return cmd
}

func runAnalyze(args []string, resultsDir, profileName string) error {
	logFile, commit, branch, network, durationStr := args[0], args[1], args[2], args[3], args[4]

	if profileName == "" {
		return fmt.Errorf("--profile is required: pass a path to a .node-bench.toml file or a directory containing one\n  reference profiles: profiles/kernel-node.toml  profiles/btcd.toml")
	}
	prof, err := nodeprofile.Resolve(profileName)
	if err != nil {
		return err
	}

	durationS, err := strconv.Atoi(durationStr)
	if err != nil {
		return fmt.Errorf("duration-s must be an integer, got %q", durationStr)
	}

	f, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	pr, err := logparse.Parse(f, prof)
	if err != nil {
		return fmt.Errorf("parse log: %w", err)
	}

	r := &result.Result{
		Commit:          commit,
		Branch:          branch,
		Network:         network,
		DurationS:       durationS,
		RunAt:           time.Now().UTC().Truncate(time.Second),
		Trials:          1,
		HeaderSyncS:     pr.HeaderSyncS,
		BlockSyncS:      pr.BlockSyncS,
		TotalElapsedS:   pr.TotalElapsedS,
		BlocksValidated: pr.BlocksValidated,
		MaxHeight:       pr.MaxHeight,
		BlocksPerSec:    pr.BlocksPerSec,
		Checkpoints:     result.Checkpoints(pr.Checkpoints),
		Machine:         result.DetectMachine(),
	}

	path, err := r.Save(resultsDir)
	if err != nil {
		return fmt.Errorf("save result: %w", err)
	}

	fmt.Printf("Result saved to %s\n", path)
	fmt.Printf("  header_sync_s=%d  block_sync_s=%d  bps=%.1f  height=%d\n",
		r.HeaderSyncS, r.BlockSyncS, r.BlocksPerSec, r.MaxHeight)
	return nil
}
