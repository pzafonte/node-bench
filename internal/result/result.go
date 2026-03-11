// Package result defines the Result type and its JSON serialization.
package result

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoints maps block height (as string key) to seconds-since-start.
// A nil pointer means that height was not reached during the run.
type Checkpoints map[string]*int

// Stats holds median and population standard deviation for a single metric
// across multiple trials.
type Stats struct {
	P50    float64 `json:"p50"`
	Stddev float64 `json:"stddev"`
}

type TrialStats struct {
	HeaderSyncS  Stats `json:"header_sync_s"`
	BlockSyncS   Stats `json:"block_sync_s"`
	BlocksPerSec Stats `json:"blocks_per_sec"`
}

// Result is the canonical record of one benchmark run (single or multi-trial).
// Its JSON representation is stored in results/<commit>.json.
type Result struct {
	Commit          string      `json:"commit"`
	Branch          string      `json:"branch"`
	Network         string      `json:"network"`
	DurationS       int         `json:"duration_s"`
	RunAt           time.Time   `json:"run_at"`
	Trials          int         `json:"trials,omitempty"`
	HeaderSyncS     int         `json:"header_sync_s"`
	BlockSyncS      int         `json:"block_sync_s"`
	TotalElapsedS   int         `json:"total_elapsed_s"`
	BlocksValidated int         `json:"blocks_validated"`
	MaxHeight       int         `json:"max_height"`
	BlocksPerSec    float64     `json:"blocks_per_sec"`
	Checkpoints     Checkpoints `json:"checkpoints"`
	Logs            []string     `json:"logs,omitempty"`
	TrialStats      *TrialStats  `json:"stats,omitempty"`
	Machine         *MachineInfo `json:"machine,omitempty"`
}

// Save writes r as indented JSON to resultsDir/<commit>.json.
func (r *Result) Save(resultsDir string) (string, error) {
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return "", fmt.Errorf("create results dir: %w", err)
	}
	path := filepath.Join(resultsDir, r.Commit+".json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write result: %w", err)
	}
	return path, nil
}

// Load reads and parses the JSON file at path into a Result.
func Load(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &r, nil
}
