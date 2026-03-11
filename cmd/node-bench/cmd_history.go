package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pzafonte/node-bench/internal/result"
	"github.com/spf13/cobra"
)

func cmdHistory() *cobra.Command {
	var resultsDir string
	cmd := &cobra.Command{
		Use:   "history",
		Short: "List all stored results sorted by most recent run",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistory(cmd.OutOrStdout(), resultsDir)
		},
	}
	cmd.Flags().StringVar(&resultsDir, "results-dir", "results",
		"directory containing result JSON files")
	return cmd
}

func runHistory(w io.Writer, resultsDir string) error {
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(w, "No results yet. Run: node-bench run <node-repo>")
			return nil
		}
		return fmt.Errorf("read results dir: %w", err)
	}

	var results []*result.Result
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		r, err := result.Load(filepath.Join(resultsDir, e.Name()))
		if err != nil {
			fmt.Fprintf(w, "  warning: skipping %s: %v\n", e.Name(), err)
			continue
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Fprintln(w, "No results yet. Run: node-bench run <node-repo>")
		return nil
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].RunAt.After(results[j].RunAt)
	})

	sep := "============================================================"
	fmt.Fprintln(w, "\n"+sep)
	fmt.Fprintln(w, "  node-bench history")
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  %-10s %-24s %-8s %-12s %-12s %-10s %-8s %s\n",
		"Commit", "Branch", "Trials", "Hdr sync(s)", "Blk sync(s)", "Blk/s", "Height", "Run at")
	fmt.Fprintln(w, "  "+strings.Repeat("-", len(sep)-2))

	for _, r := range results {
		trials := "-"
		if r.Trials > 0 {
			trials = fmt.Sprintf("%d", r.Trials)
		}
		stddev := ""
		if r.TrialStats != nil {
			stddev = fmt.Sprintf(" (±%.1f)", r.TrialStats.BlocksPerSec.Stddev)
		}
		fmt.Fprintf(w, "  %-10s %-24s %-8s %-12d %-12d %-10s %-8d %s\n",
			r.Commit,
			truncate(r.Branch, 23),
			trials,
			r.HeaderSyncS,
			r.BlockSyncS,
			fmt.Sprintf("%.1f%s", r.BlocksPerSec, stddev),
			r.MaxHeight,
			r.RunAt.Format("2006-01-02 15:04"),
		)
	}

	fmt.Fprintln(w, sep+"\n")
	return nil
}

// truncate shortens s to at most n characters, adding "…" if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
