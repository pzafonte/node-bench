package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pzafonte/node-bench/internal/result"
)

// syntheticLog is a minimal log accepted by the kernel-node-style parser:
// one peer-connect line at T=0 and two UpdateTip lines at T=60s and T=120s.
const syntheticLog = `2026-01-01T00:00:00Z INFO  Connected to peer 127.0.0.1:38333
2026-01-01T00:01:00Z INFO  UpdateTip: height=1 hash=abc
2026-01-01T00:02:00Z INFO  UpdateTip: height=1000 hash=def
`

// kernelStyleToml is a minimal profile TOML for the synthetic log above.
const kernelStyleToml = `
name        = "test-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/test-node"
[logs]
connected_to_peer = "Connected to"
update_tip        = "UpdateTip"
timestamp_layout  = "2006-01-02T15:04:05Z"
`

func TestAnalyzeMissingProfile(t *testing.T) {
	err := runAnalyze([]string{"/dev/null", "abc1234", "master", "signet", "300"}, "results", "")
	if err == nil {
		t.Fatal("expected error for missing --profile, got nil")
	}
}

func TestAnalyzeBadDuration(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "test.toml")
	if err := os.WriteFile(profilePath, []byte(kernelStyleToml), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runAnalyze([]string{"/dev/null", "abc1234", "master", "signet", "notanumber"}, dir, profilePath)
	if err == nil {
		t.Fatal("expected error for non-integer duration-s, got nil")
	}
}

func TestAnalyzeUnreadableLog(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "test.toml")
	if err := os.WriteFile(profilePath, []byte(kernelStyleToml), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runAnalyze([]string{"/nonexistent/log.txt", "abc1234", "master", "signet", "300"}, dir, profilePath)
	if err == nil {
		t.Fatal("expected error for missing log file, got nil")
	}
}

func TestAnalyzeHappyPath(t *testing.T) {
	dir := t.TempDir()

	// Write the synthetic log to a temp file.
	logPath := filepath.Join(dir, "run.log")
	if err := os.WriteFile(logPath, []byte(syntheticLog), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write the profile TOML.
	profilePath := filepath.Join(dir, "test.toml")
	if err := os.WriteFile(profilePath, []byte(kernelStyleToml), 0o644); err != nil {
		t.Fatal(err)
	}

	resultsDir := filepath.Join(dir, "results")
	err := runAnalyze(
		[]string{logPath, "deadbeef", "main", "signet", "300"},
		resultsDir, profilePath,
	)
	if err != nil {
		t.Fatalf("runAnalyze: %v", err)
	}

	// Load the saved result and check its fields.
	r, err := result.Load(filepath.Join(resultsDir, "deadbeef.json"))
	if err != nil {
		t.Fatalf("load result: %v", err)
	}

	if r.Commit != "deadbeef" {
		t.Errorf("Commit = %q, want \"deadbeef\"", r.Commit)
	}
	if r.Branch != "main" {
		t.Errorf("Branch = %q, want \"main\"", r.Branch)
	}
	if r.Network != "signet" {
		t.Errorf("Network = %q, want \"signet\"", r.Network)
	}
	if r.DurationS != 300 {
		t.Errorf("DurationS = %d, want 300", r.DurationS)
	}
	// Synthetic log: T=0 connect, T=60 first block --> HeaderSyncS = 60.
	if r.HeaderSyncS != 60 {
		t.Errorf("HeaderSyncS = %d, want 60", r.HeaderSyncS)
	}
	// T=60 first block, T=120 last block --> BlockSyncS = 60.
	if r.BlockSyncS != 60 {
		t.Errorf("BlockSyncS = %d, want 60", r.BlockSyncS)
	}
	if r.TotalElapsedS != 120 {
		t.Errorf("TotalElapsedS = %d, want 120", r.TotalElapsedS)
	}
	// BlocksValidated = maxHeight (we always start from genesis).
	if r.BlocksValidated != 1000 {
		t.Errorf("BlocksValidated = %d, want 1000", r.BlocksValidated)
	}
	if r.MaxHeight != 1000 {
		t.Errorf("MaxHeight = %d, want 1000", r.MaxHeight)
	}
	// Checkpoint at height 1000 should be at 120s elapsed.
	if v := r.Checkpoints["1000"]; v == nil {
		t.Error("Checkpoints[1000] = nil, want &120")
	} else if *v != 120 {
		t.Errorf("Checkpoints[1000] = %d, want 120", *v)
	}
}
