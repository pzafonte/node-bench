package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pzafonte/node-bench/internal/result"
)

// writeResult saves r as <commit>.json in dir and returns the path.
func writeResult(t *testing.T, dir string, r *result.Result) string {
	t.Helper()
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	path := filepath.Join(dir, r.Commit+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}
	return path
}

func TestHistoryMissingDir(t *testing.T) {
	var buf bytes.Buffer
	if err := runHistory(&buf, "/tmp/node-bench-nonexistent-dir-xyz"); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	if !strings.Contains(buf.String(), "No results yet") {
		t.Errorf("expected 'No results yet', got: %s", buf.String())
	}
}

func TestHistoryEmptyDir(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := runHistory(&buf, dir); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	if !strings.Contains(buf.String(), "No results yet") {
		t.Errorf("expected 'No results yet', got: %s", buf.String())
	}
}

func TestHistorySingleResult(t *testing.T) {
	dir := t.TempDir()
	writeResult(t, dir, &result.Result{
		Commit:       "abc1234",
		Branch:       "master",
		Network:      "signet",
		Trials:       1,
		HeaderSyncS:  180,
		BlockSyncS:   110,
		BlocksPerSec: 3.2,
		MaxHeight:    5000,
		RunAt:        time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
	})

	var buf bytes.Buffer
	if err := runHistory(&buf, dir); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "abc1234") {
		t.Errorf("output missing commit: %s", out)
	}
	if !strings.Contains(out, "master") {
		t.Errorf("output missing branch: %s", out)
	}
	if !strings.Contains(out, "3.2") {
		t.Errorf("output missing bps: %s", out)
	}
}

func TestHistorySortsMostRecentFirst(t *testing.T) {
	dir := t.TempDir()

	oldest := &result.Result{
		Commit: "old0001", Branch: "v1",
		RunAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	middle := &result.Result{
		Commit: "mid0002", Branch: "v2",
		RunAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	newest := &result.Result{
		Commit: "new0003", Branch: "v3",
		RunAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	writeResult(t, dir, oldest)
	writeResult(t, dir, middle)
	writeResult(t, dir, newest)

	var buf bytes.Buffer
	if err := runHistory(&buf, dir); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out := buf.String()

	// Check that "new" appears before "mid" and "mid" before "old".
	posNew := strings.Index(out, "new0003")
	posMid := strings.Index(out, "mid0002")
	posOld := strings.Index(out, "old0001")

	if posNew < 0 || posMid < 0 || posOld < 0 {
		t.Fatalf("not all commits found in output:\n%s", out)
	}
	if posNew > posMid {
		t.Errorf("newest (%d) should appear before middle (%d)", posNew, posMid)
	}
	if posMid > posOld {
		t.Errorf("middle (%d) should appear before oldest (%d)", posMid, posOld)
	}
}

func TestHistoryShowsStddev(t *testing.T) {
	dir := t.TempDir()
	writeResult(t, dir, &result.Result{
		Commit:       "xyz9999",
		Branch:       "main",
		Trials:       5,
		BlocksPerSec: 3.5,
		RunAt:        time.Now().UTC(),
		TrialStats: &result.TrialStats{
			BlocksPerSec: result.Stats{P50: 3.5, Stddev: 0.2},
		},
	})

	var buf bytes.Buffer
	if err := runHistory(&buf, dir); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "±0.2") {
		t.Errorf("expected ±0.2 in output, got:\n%s", out)
	}
}
