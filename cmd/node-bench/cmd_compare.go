package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/pzafonte/node-bench/internal/result"
	"github.com/spf13/cobra"
)

func cmdCompare() *cobra.Command {
	var resultsDir string
	cmd := &cobra.Command{
		Use:   "compare <sha-a> <sha-b>",
		Short: "Print a side-by-side comparison of two stored results",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompare(cmd.OutOrStdout(), args[0], args[1], resultsDir)
		},
	}
	cmd.Flags().StringVar(&resultsDir, "results-dir", "results",
		"directory containing result JSON files")
	return cmd
}

func runCompare(w io.Writer, shaA, shaB, resultsDir string) error {
	pathA := filepath.Join(resultsDir, shaA+".json")
	pathB := filepath.Join(resultsDir, shaB+".json")

	rA, err := result.Load(pathA)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no result for %s (run 'analyze' or 'run' first)", shaA)
		}
		return err
	}
	rB, err := result.Load(pathB)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no result for %s (run 'analyze' or 'run' first)", shaB)
		}
		return err
	}

	printComparison(w, shaA, shaB, rA, rB)
	return nil
}

func printComparison(w io.Writer, shaA, shaB string, rA, rB *result.Result) {
	hdr := fmt.Sprintf("%-28s %-22s %-22s %s", "",
		fmt.Sprintf("%s (%s)", shaA, rA.Branch),
		fmt.Sprintf("%s (%s)", shaB, rB.Branch),
		"Δ")

	sep := "============================================================"
	fmt.Fprintln(w, "\n"+sep)
	fmt.Fprintln(w, "  node-bench comparison")
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  %s\n", hdr)

	row(w, "Header sync (s)", rA.HeaderSyncS, rB.HeaderSyncS, true)
	row(w, "Block sync (s)", rA.BlockSyncS, rB.BlockSyncS, true)
	rowF(w, "Throughput (blk/s)", rA.BlocksPerSec, rB.BlocksPerSec, false)
	row(w, "Max height", rA.MaxHeight, rB.MaxHeight, false)
	row(w, "Blocks validated", rA.BlocksValidated, rB.BlocksValidated, false)

	if rA.TrialStats != nil && rB.TrialStats != nil {
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "  %-28s %-22s %-22s\n", "Stddev (blk/s)",
			fmt.Sprintf("±%.1f", rA.TrialStats.BlocksPerSec.Stddev),
			fmt.Sprintf("±%.1f", rB.TrialStats.BlocksPerSec.Stddev))
	}

	fmt.Fprintln(w, "\n------------------------------------------------------------")
	fmt.Fprintln(w, "  Checkpoint times (seconds from node start)")
	fmt.Fprintln(w, "------------------------------------------------------------")
	fmt.Fprintf(w, "  %-10s %-22s %-22s\n", "Height", shaA, shaB)

	for _, h := range mergedCheckpointHeights(rA.Checkpoints, rB.Checkpoints) {
		key := strconv.Itoa(h)
		vA := checkpointStr(rA.Checkpoints, key)
		vB := checkpointStr(rB.Checkpoints, key)
		fmt.Fprintf(w, "  %-10d %-22s %-22s\n", h, vA, vB)
	}

	fmt.Fprintln(w, sep+"\n")
}

// row prints one integer metric row with a Δ% column.
// lowerIsBetter=true means a decrease is an improvement (shown as positive Δ).
func row(w io.Writer, label string, a, b int, lowerIsBetter bool) {
	delta := ""
	if a != 0 {
		pct := float64(b-a) / float64(a) * 100
		if lowerIsBetter {
			pct = -pct
		}
		if pct >= 0 {
			delta = fmt.Sprintf("+%.1f%%", pct)
		} else {
			delta = fmt.Sprintf("%.1f%%", pct)
		}
	}
	fmt.Fprintf(w, "  %-28s %-22d %-22d %s\n", label, a, b, delta)
}

// rowF is like row but for float64 metrics.
func rowF(w io.Writer, label string, a, b float64, lowerIsBetter bool) {
	delta := ""
	if a != 0 {
		pct := (b - a) / a * 100
		if lowerIsBetter {
			pct = -pct
		}
		if pct >= 0 {
			delta = fmt.Sprintf("+%.1f%%", pct)
		} else {
			delta = fmt.Sprintf("%.1f%%", pct)
		}
	}
	fmt.Fprintf(w, "  %-28s %-22.1f %-22.1f %s\n", label, a, b, delta)
}

// mergedCheckpointHeights returns the union of checkpoint height keys from
// both results, sorted numerically ascending.
func mergedCheckpointHeights(a, b result.Checkpoints) []int {
	seen := make(map[int]struct{})
	for k := range a {
		if h, err := strconv.Atoi(k); err == nil {
			seen[h] = struct{}{}
		}
	}
	for k := range b {
		if h, err := strconv.Atoi(k); err == nil {
			seen[h] = struct{}{}
		}
	}
	heights := make([]int, 0, len(seen))
	for h := range seen {
		heights = append(heights, h)
	}
	sort.Ints(heights)
	return heights
}

func checkpointStr(cp result.Checkpoints, key string) string {
	if v, ok := cp[key]; ok && v != nil {
		return fmt.Sprintf("%ds", *v)
	}
	return "not reached"
}
