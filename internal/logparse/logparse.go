// Package logparse extracts IBD timing metrics from a node's log output.
package logparse

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/pzafonte/node-bench/internal/nodeprofile"
)

// DefaultCheckpointHeights are the block heights recorded when a profile
// does not specify its own checkpoints list. Matches bitcoinperf's
// reporting granularity for easy comparison with mainnet IBD results.
var DefaultCheckpointHeights = []int{1000, 5000, 10000, 25000, 50000, 100000, 150000, 200000}

func checkpointHeights(prof *nodeprofile.NodeProfile) []int {
	if len(prof.Logs.Checkpoints) > 0 {
		return prof.Logs.Checkpoints
	}
	return DefaultCheckpointHeights
}

// defaultTimestampPattern matches both kernel-node ("2026-03-10T16:45:01Z") and
// btcd ("2026-03-10 16:45:01.000") timestamp prefixes.
// Used when LogPatterns.TimestampPattern is empty.
const defaultTimestampPattern = `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[\d.Z]*`

// defaultHeightPattern extracts block height from "height=N" log lines.
// Used when LogPatterns.HeightPattern is empty.
const defaultHeightPattern = `height=(\d+)`

// ParseResult holds the raw metrics extracted from a single log file.
// It maps 1:1 to the fields in result.Result that come from log parsing.
type ParseResult struct {
	HeaderSyncS     int
	BlockSyncS      int
	TotalElapsedS   int
	BlocksValidated int
	MaxHeight       int
	BlocksPerSec    float64
	// Checkpoints maps height (as string key) to seconds-since-start.
	// nil value means that height was not reached.
	Checkpoints map[string]*int
}

// Parse reads a node log from r and extracts IBD timing metrics.
// The NodeProfile supplies the regex patterns specific to that node implementation.
func Parse(r io.Reader, prof *nodeprofile.NodeProfile) (*ParseResult, error) {
	connRE, err := regexp.Compile(prof.Logs.ConnectedToPeer)
	if err != nil {
		return nil, fmt.Errorf("compile ConnectedToPeer pattern: %w", err)
	}
	tipRE, err := regexp.Compile(prof.Logs.UpdateTip)
	if err != nil {
		return nil, fmt.Errorf("compile UpdateTip pattern: %w", err)
	}

	tsPattern := prof.Logs.TimestampPattern
	if tsPattern == "" {
		tsPattern = defaultTimestampPattern
	}
	tsRE, err := regexp.Compile(tsPattern)
	if err != nil {
		return nil, fmt.Errorf("compile TimestampPattern: %w", err)
	}

	hPattern := prof.Logs.HeightPattern
	if hPattern == "" {
		hPattern = defaultHeightPattern
	}
	heightRE, err := regexp.Compile(hPattern)
	if err != nil {
		return nil, fmt.Errorf("compile HeightPattern: %w", err)
	}

	cpHeights := checkpointHeights(prof)

	var startTime, firstBlockTime, lastBlockTime time.Time
	maxHeight := 0
	checkpointTimes := make(map[int]time.Time)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		if startTime.IsZero() && connRE.MatchString(line) {
			if ts := parseTimestamp(line, tsRE, prof.Logs.TimestampLayout); !ts.IsZero() {
				startTime = ts
			}
			continue
		}

		if !tipRE.MatchString(line) {
			continue
		}
		m := heightRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		h, err := strconv.Atoi(m[1])
		if err != nil || h <= 0 {
			continue
		}
		ts := parseTimestamp(line, tsRE, prof.Logs.TimestampLayout)
		if ts.IsZero() {
			continue
		}

		if firstBlockTime.IsZero() {
			firstBlockTime = ts
		}
		lastBlockTime = ts
		if h > maxHeight {
			maxHeight = h
		}
		for _, cp := range cpHeights {
			if h >= cp {
				if _, seen := checkpointTimes[cp]; !seen {
					checkpointTimes[cp] = ts
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan log: %w", err)
	}

	pr := &ParseResult{
		// Use maxHeight as the block count. We always start from genesis,
		// so maxHeight == total blocks processed for both per-block loggers
		// (kernel-node) and batch-progress loggers (btcd).
		BlocksValidated: maxHeight,
		MaxHeight:       maxHeight,
		Checkpoints:     make(map[string]*int),
	}

	if !startTime.IsZero() && !lastBlockTime.IsZero() {
		elapsed := int(lastBlockTime.Sub(startTime).Seconds())
		pr.TotalElapsedS = elapsed
		if !firstBlockTime.IsZero() {
			pr.HeaderSyncS = int(firstBlockTime.Sub(startTime).Seconds())
			pr.BlockSyncS = int(lastBlockTime.Sub(firstBlockTime).Seconds())
		}
		// Use maxHeight/elapsed for BPS. For per-block loggers (kernel-node)
		// maxHeight == blocksValidated so the result is the same. For batch
		// loggers (btcd) blocksValidated is the batch count, not block count,
		// so maxHeight gives the accurate rate.
		if elapsed > 0 && maxHeight > 0 {
			pr.BlocksPerSec = math.Round(float64(maxHeight)/float64(elapsed)*10) / 10
		}
	}

	for _, cp := range cpHeights {
		key := strconv.Itoa(cp)
		if t, ok := checkpointTimes[cp]; ok && !startTime.IsZero() {
			secs := int(t.Sub(startTime).Seconds())
			pr.Checkpoints[key] = &secs
		} else {
			pr.Checkpoints[key] = nil
		}
	}

	return pr, nil
}

// parseTimestamp extracts the first match of re from line, then parses it with layout.
// Returns the zero time if not found or unparseable.
func parseTimestamp(line string, re *regexp.Regexp, layout string) time.Time {
	m := re.FindString(line)
	if m == "" {
		return time.Time{}
	}
	t, err := time.Parse(layout, m)
	if err != nil {
		return time.Time{}
	}
	return t
}
