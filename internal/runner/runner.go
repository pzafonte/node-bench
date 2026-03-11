// Package runner builds a node from source, executes timed benchmark trials,
// and aggregates per-trial log output into a structured Result.
package runner

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pzafonte/node-bench/internal/logparse"
	"github.com/pzafonte/node-bench/internal/nodeprofile"
	"github.com/pzafonte/node-bench/internal/result"
)

// Config holds everything the runner needs to execute a benchmark.
type Config struct {
	// RepoPath is the absolute path to the node's git repository.
	RepoPath string
	Profile  *nodeprofile.NodeProfile
	// Network is the Bitcoin network name passed to the node (e.g. "signet").
	Network  string
	// Duration is how long to run the node per trial.
	Duration time.Duration
	Trials   int
	// Peers is the pool of peer addresses to connect to.
	// Each trial picks one at random. Empty means no --connect flag.
	Peers []string
	// LogsDir is where per-trial log files are written.
	LogsDir string
}

// Run builds the node, executes Trials benchmark runs, and returns the
// aggregated Result. It does not save the result — that is the caller's job.
func Run(cfg Config) (*result.Result, error) {
	commit, err := gitRevShort(cfg.RepoPath)
	if err != nil {
		return nil, err
	}
	branch, err := gitBranch(cfg.RepoPath)
	if err != nil {
		return nil, err
	}

	if err := build(cfg.RepoPath, cfg.Profile); err != nil {
		return nil, err
	}

	binary := filepath.Join(cfg.RepoPath, cfg.Profile.BinaryPath)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	dataDir := filepath.Join(homeDir, ".node-bench-"+commit)

	// Remove chain data on exit. Log files in cfg.LogsDir are preserved.
	defer os.RemoveAll(dataDir)

	if err := os.MkdirAll(cfg.LogsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}

	var logFiles []string
	for trial := 1; trial <= cfg.Trials; trial++ {
		logFile := trialLogPath(cfg.LogsDir, commit, trial, cfg.Trials)
		peer := pickPeer(cfg.Peers)

		fmt.Printf("Trial %d/%d  commit=%s  duration=%ds  network=%s",
			trial, cfg.Trials, commit, int(cfg.Duration.Seconds()), cfg.Network)
		if peer != "" {
			fmt.Printf("  peer=%s", peer)
		}
		fmt.Println()

		if err := runTrial(cfg, binary, dataDir, logFile, peer); err != nil {
			return nil, fmt.Errorf("trial %d: %w", trial, err)
		}
		fmt.Printf("  log: %s\n", logFile)
		logFiles = append(logFiles, logFile)
	}

	return aggregate(logFiles, cfg.Profile, commit, branch, cfg.Network, int(cfg.Duration.Seconds()))
}

func build(repoPath string, prof *nodeprofile.NodeProfile) error {
	fmt.Printf("Building %s (%s)...\n", prof.Name, strings.Join(prof.BuildCmd, " "))
	cmd := exec.Command(prof.BuildCmd[0], prof.BuildCmd[1:]...)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stderr // build output goes to stderr so stdout stays clean
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build %s: %w", prof.Name, err)
	}
	return nil
}

func runTrial(cfg Config, binary, dataDir, logFile, peer string) error {
	// Fresh datadir for each trial so we start from genesis every time.
	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("clean datadir: %w", err)
	}

	var args []string
	switch {
	case cfg.Profile.Flags.NetworkFlag != "":
		args = append(args, cfg.Profile.Flags.NetworkFlag)
	case cfg.Profile.Flags.Network != "":
		args = append(args, cfg.Profile.Flags.Network, cfg.Network)
	}
	args = append(args, cfg.Profile.Flags.Datadir, dataDir)
	args = append(args, cfg.Profile.ExtraArgs...)
	if peer != "" {
		args = append(args, cfg.Profile.Flags.Connect, peer)
	}

	log, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}
	defer log.Close()

	cmd := exec.Command(binary, args...)
	cmd.Stdout = log
	cmd.Stderr = log

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start node: %w", err)
	}

	time.Sleep(cfg.Duration)

	// Kill the node. We don't need graceful shutdown — the log is already on disk.
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	return nil
}

func aggregate(logFiles []string, prof *nodeprofile.NodeProfile, commit, branch, network string, durationS int) (*result.Result, error) {
	var trials []logparse.ParseResult
	for _, lf := range logFiles {
		f, err := os.Open(lf)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", lf, err)
		}
		pr, err := logparse.Parse(f, prof)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", lf, err)
		}
		trials = append(trials, *pr)
	}

	n := len(trials)
	if n == 0 {
		return nil, fmt.Errorf("no trial results to aggregate")
	}

	hdrV := make([]float64, n)
	bsyncV := make([]float64, n)
	bpsV := make([]float64, n)
	elapV := make([]float64, n)
	blksV := make([]float64, n)
	maxHeight := 0

	for i, t := range trials {
		hdrV[i] = float64(t.HeaderSyncS)
		bsyncV[i] = float64(t.BlockSyncS)
		bpsV[i] = t.BlocksPerSec
		elapV[i] = float64(t.TotalElapsedS)
		blksV[i] = float64(t.BlocksValidated)
		if t.MaxHeight > maxHeight {
			maxHeight = t.MaxHeight
		}
	}

	checkpoints := make(result.Checkpoints)
	cpHeights := logparse.DefaultCheckpointHeights
	if len(prof.Logs.Checkpoints) > 0 {
		cpHeights = prof.Logs.Checkpoints
	}
	for _, cp := range cpHeights {
		key := strconv.Itoa(cp)
		var vals []float64
		for _, t := range trials {
			if v, ok := t.Checkpoints[key]; ok && v != nil {
				vals = append(vals, float64(*v))
			}
		}
		if len(vals) > 0 {
			m := int(medianF(vals))
			checkpoints[key] = &m
		} else {
			checkpoints[key] = nil
		}
	}

	r := &result.Result{
		Commit:          commit,
		Branch:          branch,
		Network:         network,
		DurationS:       durationS,
		RunAt:           time.Now().UTC().Truncate(time.Second),
		Trials:          n,
		HeaderSyncS:     int(medianF(hdrV)),
		BlockSyncS:      int(medianF(bsyncV)),
		TotalElapsedS:   int(medianF(elapV)),
		BlocksValidated: int(medianF(blksV)),
		MaxHeight:       maxHeight,
		BlocksPerSec:    round1(medianF(bpsV)),
		Checkpoints:     checkpoints,
		Logs:            logFiles,
		Machine:         result.DetectMachine(),
	}

	if n > 1 {
		r.TrialStats = &result.TrialStats{
			HeaderSyncS:  result.Stats{P50: medianF(hdrV), Stddev: round1(pstdev(hdrV))},
			BlockSyncS:   result.Stats{P50: medianF(bsyncV), Stddev: round1(pstdev(bsyncV))},
			BlocksPerSec: result.Stats{P50: round1(medianF(bpsV)), Stddev: round1(pstdev(bpsV))},
		}
	}

	return r, nil
}

// trialLogPath returns the log file path for a given trial.
// Single-trial runs use <commit>.log; multi-trial use <commit>_<N>.log.
func trialLogPath(logsDir, commit string, trial, totalTrials int) string {
	if totalTrials == 1 {
		return filepath.Join(logsDir, commit+".log")
	}
	return filepath.Join(logsDir, fmt.Sprintf("%s_%d.log", commit, trial))
}

func pickPeer(peers []string) string {
	if len(peers) == 0 {
		return ""
	}
	return peers[rand.Intn(len(peers))]
}

func gitRevShort(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// gitBranch returns the current branch name, or the short SHA if HEAD is detached.
func gitBranch(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return gitRevShort(dir)
	}
	return branch, nil
}

// medianF returns the median of vals, or 0 for an empty slice.
func medianF(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// pstdev returns the population standard deviation of vals.
func pstdev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	mean := sum / float64(len(vals))
	var variance float64
	for _, v := range vals {
		d := v - mean
		variance += d * d
	}
	return math.Sqrt(variance / float64(len(vals)))
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
