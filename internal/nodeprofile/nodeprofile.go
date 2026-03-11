// Package nodeprofile loads and validates node implementation profiles from TOML files.
package nodeprofile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// LogPatterns holds the regex patterns for extracting IBD timing from a node's log.
type LogPatterns struct {
	// ConnectedToPeer matches the line a node logs when it first connects
	// to a peer and begins syncing. The timestamp must be in the line.
	ConnectedToPeer string `toml:"connected_to_peer"`

	// UpdateTip matches the line logged each time the node extends its
	// best chain by one block. The line must also match HeightPattern.
	UpdateTip string `toml:"update_tip"`

	// TimestampLayout is the Go time.Parse layout for timestamps in log lines.
	// Use "2006-01-02T15:04:05Z" for RFC3339, "2006-01-02T15:04:05.000" for ms.
	TimestampLayout string `toml:"timestamp_layout"`

	// TimestampPattern is a regex that finds a timestamp string within a log line.
	// The full match is passed to time.Parse with TimestampLayout.
	// Default (if empty): `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[\d.Z]*`
	TimestampPattern string `toml:"timestamp_pattern"`

	// HeightPattern is a regex with exactly one capture group that extracts
	// the block height (as digits) from an UpdateTip line.
	// Default (if empty): `height=(\d+)`
	HeightPattern string `toml:"height_pattern"`

	// Checkpoints lists block heights at which elapsed time since sync
	// start is recorded in the result. Omit to use the default set:
	// [1000, 5000, 10000, 25000, 50000, 100000, 150000, 200000]
	Checkpoints []int `toml:"checkpoints"`
}

// NodeFlags holds the CLI flag names for options that node-bench controls.
// Different implementations may use different flag names for the same concept.
type NodeFlags struct {
	// Network is the flag name for value-style network selection.
	// Used as: <Network> <network-value>  e.g. "--network signet"
	// Ignored when NetworkFlag is set.
	Network string `toml:"network"`

	// NetworkFlag, when non-empty, is appended verbatim to node args for
	// network selection. Use this for nodes that use boolean flags like
	// --signet instead of --network signet.
	// Example: "--signet"
	NetworkFlag string `toml:"network_flag"`

	Datadir string `toml:"datadir"` // e.g. "--datadir"
	Connect string `toml:"connect"` // e.g. "--connect"
}

// NodeProfile describes everything node-bench needs to know about a specific
// Bitcoin node implementation: how to build it, how to launch it, and how to
// read its log output.
//
// Profiles are loaded from .node-bench.toml files. See profiles/ in the
// node-bench repo for reference profiles (kernel-node, btcd).
type NodeProfile struct {
	// Name is a short human-readable identifier, e.g. "kernel-node" or "btcd".
	Name string `toml:"name"`

	// BuildCmd is the command (and arguments) used to compile the node from
	// source inside the repository directory.
	// Example: ["cargo", "build", "--release"]
	// Example: ["go", "build", "./cmd/btcd/"]
	BuildCmd []string `toml:"build_cmd"`

	// BinaryPath is the path to the compiled binary, relative to the repo root.
	// Example: "target/release/node"
	// Example: "btcd"
	BinaryPath string `toml:"binary_path"`

	Flags NodeFlags  `toml:"flags"`
	Logs  LogPatterns `toml:"logs"`

	// DefaultPeers is the pool of peer addresses used when --connect is not
	// provided on the command line. Each trial picks one at random.
	// Example: ["45.94.168.5:38333", "208.68.4.50:38333"]
	DefaultPeers []string `toml:"default_peers"`

	// ExtraArgs is a list of additional arguments passed verbatim to the
	// node binary on every trial, after the network and datadir flags but
	// before the connect flag. Use for node-specific options that node-bench
	// does not control directly.
	// Example: ["--nolisten"] to disable inbound connections in btcd.
	ExtraArgs []string `toml:"extra_args"`
}

// Resolve returns the NodeProfile for the given name.
//
// Resolution order:
//  1. If name is an existing directory, load .node-bench.toml from inside it.
//  2. If name looks like a file path (absolute, ./ ../ or .toml suffix), load it.
//  3. Otherwise return an error — there are no built-in profiles.
func Resolve(name string) (*NodeProfile, error) {
	if fi, err := os.Stat(name); err == nil && fi.IsDir() {
		return LoadFromFile(filepath.Join(name, ".node-bench.toml"))
	}
	if isFilePath(name) {
		return LoadFromFile(name)
	}
	return nil, fmt.Errorf("profile %q is not a file path or directory containing .node-bench.toml", name)
}

// LoadFromFile reads and validates a NodeProfile from a TOML file.
// See profiles/ in the node-bench repo for reference profiles.
func LoadFromFile(path string) (*NodeProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile %s: %w", path, err)
	}
	var p NodeProfile
	if err := toml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile %s: %w", path, err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("profile %s: missing required field \"name\"", path)
	}
	if len(p.BuildCmd) == 0 {
		return nil, fmt.Errorf("profile %s: missing required field \"build_cmd\"", path)
	}
	if p.BinaryPath == "" {
		return nil, fmt.Errorf("profile %s: missing required field \"binary_path\"", path)
	}
	return &p, nil
}

// isFilePath reports whether name looks like a file path rather than a profile name.
func isFilePath(name string) bool {
	return strings.HasPrefix(name, "/") ||
		strings.HasPrefix(name, "./") ||
		strings.HasPrefix(name, "../") ||
		strings.HasSuffix(name, ".toml")
}
