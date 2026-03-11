package runner

import (
	"math"
	"os"
	"testing"

	"github.com/pzafonte/node-bench/internal/nodeprofile"
)

func TestMedianF(t *testing.T) {
	tests := []struct {
		name string
		vals []float64
		want float64
	}{
		{"single element", []float64{42}, 42},
		{"odd count", []float64{3, 1, 2}, 2},
		{"even count", []float64{1, 2, 3, 4}, 2.5},
		{"already sorted", []float64{10, 20, 30}, 20},
		{"reverse sorted", []float64{30, 20, 10}, 20},
		{"with duplicates", []float64{5, 5, 5, 5}, 5},
		{"empty returns 0", []float64{}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := medianF(tc.vals)
			if got != tc.want {
				t.Errorf("medianF(%v) = %v, want %v", tc.vals, got, tc.want)
			}
		})
	}
}

func TestPstdev(t *testing.T) {
	tests := []struct {
		name string
		vals []float64
		want float64 // rounded to 2 decimal places for comparison
	}{
		{"single element returns 0", []float64{42}, 0},
		{"two equal values", []float64{5, 5}, 0},
		{"two values", []float64{2, 4}, 1},
		// Wikipedia example for population stddev: https://en.wikipedia.org/wiki/Standard_deviation
		{"known result", []float64{2, 4, 4, 4, 5, 5, 7, 9}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := math.Round(pstdev(tc.vals)*100) / 100
			if got != tc.want {
				t.Errorf("pstdev(%v) = %v, want %v", tc.vals, got, tc.want)
			}
		})
	}
}

func TestTrialLogPath(t *testing.T) {
	tests := []struct {
		total int
		trial int
		want  string
	}{
		{1, 1, "logs/abc1234.log"},
		{5, 1, "logs/abc1234_1.log"},
		{5, 3, "logs/abc1234_3.log"},
		{5, 5, "logs/abc1234_5.log"},
	}
	for _, tc := range tests {
		got := trialLogPath("logs", "abc1234", tc.trial, tc.total)
		if got != tc.want {
			t.Errorf("trialLogPath(total=%d, trial=%d) = %q, want %q",
				tc.total, tc.trial, got, tc.want)
		}
	}
}

// TestAggregateIntegration is skipped when the log files are absent.
// To run: place matching logs at logs/e3f7fa1_{1..5}.log and a profile at
// profiles/kernel-node.toml (or adjust the paths below).
func TestAggregateIntegration(t *testing.T) {
	logFiles := []string{
		"../../logs/e3f7fa1_1.log",
		"../../logs/e3f7fa1_2.log",
		"../../logs/e3f7fa1_3.log",
		"../../logs/e3f7fa1_4.log",
		"../../logs/e3f7fa1_5.log",
	}
	for _, f := range logFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Skipf("real log files not available: %s", f)
		}
	}

	prof, err := nodeprofile.LoadFromFile("../../profiles/kernel-node.toml")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	r, err := aggregate(logFiles, prof, "e3f7fa1", "e3f7fa1", "signet", 300)
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}

	if r.Trials != 5 {
		t.Errorf("Trials = %d, want 5", r.Trials)
	}
	if r.HeaderSyncS <= 0 {
		t.Errorf("HeaderSyncS = %d, want > 0", r.HeaderSyncS)
	}
	if r.BlockSyncS <= 0 {
		t.Errorf("BlockSyncS = %d, want > 0", r.BlockSyncS)
	}
	if r.BlocksPerSec <= 0 {
		t.Errorf("BlocksPerSec = %.1f, want > 0", r.BlocksPerSec)
	}
	if r.HeaderSyncS+r.BlockSyncS != r.TotalElapsedS {
		t.Errorf("HeaderSyncS(%d) + BlockSyncS(%d) = %d, want TotalElapsedS(%d)",
			r.HeaderSyncS, r.BlockSyncS, r.HeaderSyncS+r.BlockSyncS, r.TotalElapsedS)
	}
	if r.TrialStats == nil {
		t.Fatal("TrialStats is nil for 5-trial run")
	}

	t.Logf("header_sync_s=%d (±%.1f)  block_sync_s=%d (±%.1f)  bps=%.1f (±%.1f)  height=%d",
		r.HeaderSyncS, r.TrialStats.HeaderSyncS.Stddev,
		r.BlockSyncS, r.TrialStats.BlockSyncS.Stddev,
		r.BlocksPerSec, r.TrialStats.BlocksPerSec.Stddev,
		r.MaxHeight)
}
