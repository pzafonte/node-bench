package result

import (
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()

	cp1000 := 42
	want := &Result{
		Commit:          "abc1234",
		Branch:          "master",
		Network:         "signet",
		DurationS:       300,
		RunAt:           time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		Trials:          3,
		HeaderSyncS:     45,
		BlockSyncS:      210,
		TotalElapsedS:   255,
		BlocksValidated: 820,
		MaxHeight:       833,
		BlocksPerSec:    3.2,
		Checkpoints:     Checkpoints{"1000": &cp1000, "5000": nil},
		Logs:            []string{"logs/abc1234_1.log", "logs/abc1234_2.log"},
		TrialStats: &TrialStats{
			HeaderSyncS:  Stats{P50: 45, Stddev: 2.1},
			BlockSyncS:   Stats{P50: 210, Stddev: 5.3},
			BlocksPerSec: Stats{P50: 3.2, Stddev: 0.1},
		},
	}

	path, err := want.Save(dir)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Logf("saved to %s", path)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Commit != want.Commit {
		t.Errorf("Commit = %q, want %q", got.Commit, want.Commit)
	}
	if got.Branch != want.Branch {
		t.Errorf("Branch = %q, want %q", got.Branch, want.Branch)
	}
	if got.Network != want.Network {
		t.Errorf("Network = %q, want %q", got.Network, want.Network)
	}
	if got.DurationS != want.DurationS {
		t.Errorf("DurationS = %d, want %d", got.DurationS, want.DurationS)
	}
	if !got.RunAt.Equal(want.RunAt) {
		t.Errorf("RunAt = %v, want %v", got.RunAt, want.RunAt)
	}
	if got.Trials != want.Trials {
		t.Errorf("Trials = %d, want %d", got.Trials, want.Trials)
	}
	if got.HeaderSyncS != want.HeaderSyncS {
		t.Errorf("HeaderSyncS = %d, want %d", got.HeaderSyncS, want.HeaderSyncS)
	}
	if got.BlockSyncS != want.BlockSyncS {
		t.Errorf("BlockSyncS = %d, want %d", got.BlockSyncS, want.BlockSyncS)
	}
	if got.TotalElapsedS != want.TotalElapsedS {
		t.Errorf("TotalElapsedS = %d, want %d", got.TotalElapsedS, want.TotalElapsedS)
	}
	if got.BlocksValidated != want.BlocksValidated {
		t.Errorf("BlocksValidated = %d, want %d", got.BlocksValidated, want.BlocksValidated)
	}
	if got.BlocksPerSec != want.BlocksPerSec {
		t.Errorf("BlocksPerSec = %.1f, want %.1f", got.BlocksPerSec, want.BlocksPerSec)
	}
	if v := got.Checkpoints["1000"]; v == nil {
		t.Error("Checkpoints[1000] = nil, want &42")
	} else if *v != cp1000 {
		t.Errorf("Checkpoints[1000] = %d, want %d", *v, cp1000)
	}
	// Checkpoint not reached — nil pointer must survive the JSON round-trip.
	if v, ok := got.Checkpoints["5000"]; !ok {
		t.Error("Checkpoints[5000] key missing after round-trip")
	} else if v != nil {
		t.Errorf("Checkpoints[5000] = %v, want nil", v)
	}
	if got.TrialStats == nil {
		t.Fatal("TrialStats = nil, want non-nil")
	}
	if got.TrialStats.BlocksPerSec.P50 != want.TrialStats.BlocksPerSec.P50 {
		t.Errorf("TrialStats.BlocksPerSec.P50 = %.1f, want %.1f",
			got.TrialStats.BlocksPerSec.P50, want.TrialStats.BlocksPerSec.P50)
	}
}

func TestDetectMachine(t *testing.T) {
	m := DetectMachine()
	if m == nil {
		t.Fatal("DetectMachine returned nil")
	}
	if m.OS == "" {
		t.Error("OS is empty")
	}
	if m.CPUs <= 0 {
		t.Errorf("CPUs = %d, want > 0", m.CPUs)
	}
	t.Logf("os=%s cpus=%d", m.OS, m.CPUs)
}

func TestMachineRoundTrip(t *testing.T) {
	// Machine field must survive a JSON save/load cycle.
	dir := t.TempDir()
	r := &Result{
		Commit:  "machtest",
		Machine: &MachineInfo{OS: "linux", CPUs: 8},
	}
	path, err := r.Save(dir)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Machine == nil {
		t.Fatal("Machine = nil after round-trip")
	}
	if got.Machine.OS != "linux" {
		t.Errorf("OS = %q, want \"linux\"", got.Machine.OS)
	}
	if got.Machine.CPUs != 8 {
		t.Errorf("CPUs = %d, want 8", got.Machine.CPUs)
	}
}
