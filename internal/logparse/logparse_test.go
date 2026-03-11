package logparse

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/pzafonte/node-bench/internal/nodeprofile"
)

// TestParseIntegration is skipped when the log file is absent.
// To run: place a node log at logs/e3f7fa1_1.log and a matching profile at
// profiles/kernel-node.toml (or adjust the paths below).
func TestParseIntegration(t *testing.T) {
	f, err := os.Open("../../logs/e3f7fa1_1.log")
	if err != nil {
		t.Skipf("real log not available: %v", err)
	}
	defer f.Close()

	prof, err := nodeprofile.LoadFromFile("../../profiles/kernel-node.toml")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	pr, err := Parse(f, prof)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if pr.BlocksValidated == 0 {
		t.Error("BlocksValidated = 0, expected > 0")
	}
	if pr.MaxHeight == 0 {
		t.Error("MaxHeight = 0, expected > 0")
	}
	if pr.TotalElapsedS == 0 {
		t.Error("TotalElapsedS = 0, expected > 0")
	}
	if pr.BlocksPerSec <= 0 {
		t.Errorf("BlocksPerSec = %.1f, expected > 0", pr.BlocksPerSec)
	}

	t.Logf("header_sync_s=%d block_sync_s=%d total_elapsed_s=%d blocks=%d height=%d bps=%.1f",
		pr.HeaderSyncS, pr.BlockSyncS, pr.TotalElapsedS,
		pr.BlocksValidated, pr.MaxHeight, pr.BlocksPerSec)

	for _, h := range DefaultCheckpointHeights {
		key := strconv.Itoa(h)
		if pr.Checkpoints[key] != nil {
			t.Logf("checkpoint %d = %ds", h, *pr.Checkpoints[key])
		}
	}

	if pr.HeaderSyncS+pr.BlockSyncS != pr.TotalElapsedS {
		t.Errorf("HeaderSyncS(%d) + BlockSyncS(%d) = %d, want TotalElapsedS(%d)",
			pr.HeaderSyncS, pr.BlockSyncS, pr.HeaderSyncS+pr.BlockSyncS, pr.TotalElapsedS)
	}
}

// TestParseSynthetic exercises the parser with a constructed log:
// T=0 connected, T=60s first block (height=1), T=120s (height=1000).
func TestParseSynthetic(t *testing.T) {
	log := strings.Join([]string{
		"2026-01-01T00:00:00Z INFO  net: Connected to peer 127.0.0.1:38333",
		"2026-01-01T00:00:05Z INFO  some other message",
		"2026-01-01T00:01:00Z INFO  chain: UpdateTip: height=1 hash=abc",
		"2026-01-01T00:02:00Z INFO  chain: UpdateTip: height=1000 hash=def",
	}, "\n")

	prof := &nodeprofile.NodeProfile{
		Logs: nodeprofile.LogPatterns{
			ConnectedToPeer: `Connected to`,
			UpdateTip:       `UpdateTip`,
			TimestampLayout: "2006-01-02T15:04:05Z",
		},
	}

	pr, err := Parse(strings.NewReader(log), prof)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if pr.HeaderSyncS != 60 {
		t.Errorf("HeaderSyncS = %d, want 60", pr.HeaderSyncS)
	}
	if pr.BlockSyncS != 60 {
		t.Errorf("BlockSyncS = %d, want 60", pr.BlockSyncS)
	}
	if pr.TotalElapsedS != 120 {
		t.Errorf("TotalElapsedS = %d, want 120", pr.TotalElapsedS)
	}
	// maxHeight (1000), not line count (2).
	if pr.BlocksValidated != 1000 {
		t.Errorf("BlocksValidated = %d, want 1000", pr.BlocksValidated)
	}
	if pr.MaxHeight != 1000 {
		t.Errorf("MaxHeight = %d, want 1000", pr.MaxHeight)
	}
	if v := pr.Checkpoints["1000"]; v == nil {
		t.Error("Checkpoints[1000] = nil, want &120")
	} else if *v != 120 {
		t.Errorf("Checkpoints[1000] = %d, want 120", *v)
	}
	if v := pr.Checkpoints["5000"]; v != nil {
		t.Errorf("Checkpoints[5000] = %d, want nil (not reached)", *v)
	}
}

// TestParseBatchProgress exercises the btcd-style batch-progress path where a
// single log line covers thousands of blocks ("Processed N blocks in the last 10s").
func TestParseBatchProgress(t *testing.T) {
	// T=0 connect, T=90s first batch (height 22253), T=100s second batch (height 37715).
	log := strings.Join([]string{
		"2026-01-01 00:00:00.000 [INF] SYNC: Syncing to block height 295063 from peer 1.2.3.4:38333",
		"2026-01-01 00:01:30.000 [INF] SYNC: Processed 22253 blocks in the last 10s (0 transactions, height 22253, ~90 MiB cache)",
		"2026-01-01 00:01:40.000 [INF] SYNC: Processed 15462 blocks in the last 10s (0 transactions, height 37715, ~95 MiB cache)",
	}, "\n")

	prof := &nodeprofile.NodeProfile{
		Logs: nodeprofile.LogPatterns{
			ConnectedToPeer:  `Syncing to block height`,
			UpdateTip:        `Processed \d+ blocks in the last`,
			TimestampLayout:  "2006-01-02 15:04:05.000",
			TimestampPattern: `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}`,
			HeightPattern:    `height (\d+)`,
		},
	}

	pr, err := Parse(strings.NewReader(log), prof)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if pr.HeaderSyncS != 90 {
		t.Errorf("HeaderSyncS = %d, want 90", pr.HeaderSyncS)
	}
	if pr.BlockSyncS != 10 {
		t.Errorf("BlockSyncS = %d, want 10", pr.BlockSyncS)
	}
	if pr.TotalElapsedS != 100 {
		t.Errorf("TotalElapsedS = %d, want 100", pr.TotalElapsedS)
	}
	if pr.MaxHeight != 37715 {
		t.Errorf("MaxHeight = %d, want 37715", pr.MaxHeight)
	}
	if pr.BlocksValidated != 37715 {
		t.Errorf("BlocksValidated = %d, want 37715", pr.BlocksValidated)
	}
	if pr.BlocksPerSec != 377.2 {
		t.Errorf("BlocksPerSec = %.1f, want 377.2", pr.BlocksPerSec)
	}
	// Checkpoints below the first batch height are all recorded at T=90s.
	for _, cp := range []int{1000, 5000, 10000} {
		key := strconv.Itoa(cp)
		v := pr.Checkpoints[key]
		if v == nil {
			t.Errorf("Checkpoints[%d] = nil, want 90", cp)
		} else if *v != 90 {
			t.Errorf("Checkpoints[%d] = %d, want 90", cp, *v)
		}
	}
	if v := pr.Checkpoints["25000"]; v == nil {
		t.Error("Checkpoints[25000] = nil, want 100")
	} else if *v != 100 {
		t.Errorf("Checkpoints[25000] = %d, want 100", *v)
	}
	if v := pr.Checkpoints["50000"]; v != nil {
		t.Errorf("Checkpoints[50000] = %d, want nil", *v)
	}
}

// TestParseEmpty verifies the parser does not panic or error on an empty log.
func TestParseEmpty(t *testing.T) {
	prof := &nodeprofile.NodeProfile{
		Logs: nodeprofile.LogPatterns{
			ConnectedToPeer: `Connected to`,
			UpdateTip:       `UpdateTip`,
			TimestampLayout: "2006-01-02T15:04:05Z",
		},
	}
	pr, err := Parse(strings.NewReader(""), prof)
	if err != nil {
		t.Fatalf("Parse on empty input: %v", err)
	}
	if pr.BlocksValidated != 0 {
		t.Errorf("BlocksValidated = %d on empty input, want 0", pr.BlocksValidated)
	}
	if pr.TotalElapsedS != 0 {
		t.Errorf("TotalElapsedS = %d on empty input, want 0", pr.TotalElapsedS)
	}
}
