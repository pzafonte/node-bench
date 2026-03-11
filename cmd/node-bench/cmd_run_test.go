package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProfileNoToml(t *testing.T) {
	// No .node-bench.toml and no --profile flag --> must error (no built-in fallback).
	dir := t.TempDir()
	_, err := resolveProfile("", dir)
	if err == nil {
		t.Fatal("expected error when no profile and no .node-bench.toml, got nil")
	}
}

func TestResolveProfileFromToml(t *testing.T) {
	dir := t.TempDir()
	toml := `
name        = "my-test-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/my-test-node"
[logs]
update_tip = "NewBestBlock"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := resolveProfile("", dir)
	if err != nil {
		t.Fatalf("resolveProfile: %v", err)
	}
	if p.Name != "my-test-node" {
		t.Errorf("Name = %q, want \"my-test-node\"", p.Name)
	}
}

func TestResolveProfileExplicitFlagWins(t *testing.T) {
	// Explicit --profile file path overrides the .node-bench.toml in the repo dir.
	dir := t.TempDir()
	autoToml := `
name        = "auto-detected"
build_cmd   = ["echo", "auto"]
binary_path = "bin/auto"
[logs]
update_tip = "AutoTip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(autoToml), 0o644); err != nil {
		t.Fatal(err)
	}
	// Write a second TOML to pass explicitly via --profile.
	explicit := filepath.Join(dir, "explicit.toml")
	explicitToml := `
name        = "explicit-node"
build_cmd   = ["echo", "explicit"]
binary_path = "bin/explicit"
[logs]
update_tip = "ExplicitTip"
`
	if err := os.WriteFile(explicit, []byte(explicitToml), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := resolveProfile(explicit, dir)
	if err != nil {
		t.Fatalf("resolveProfile: %v", err)
	}
	if p.Name != "explicit-node" {
		t.Errorf("Name = %q, want \"explicit-node\" (explicit flag must win)", p.Name)
	}
}

func TestRunRequiresPeers(t *testing.T) {
	// No --connect and no default_peers in profile --> must fail with a clear error.
	dir := t.TempDir()
	toml := `
name        = "test-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/test-node"
[logs]
update_tip = "Tip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	root := buildRootCmd()
	root.SetArgs([]string{"run", dir})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no peers configured, got nil")
	}
}

func TestRunUsesProfileDefaultPeers(t *testing.T) {
	// Profile with default_peers and no --connect --> proceeds past peer check
	// (will fail later trying to build the node, which is expected in a unit test).
	dir := t.TempDir()
	toml := `
name        = "test-node"
build_cmd   = ["false"]
binary_path = "bin/test-node"
default_peers = ["127.0.0.1:38333"]
[logs]
update_tip = "Tip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	root := buildRootCmd()
	root.SetArgs([]string{"run", dir})
	err := root.Execute()
	// The command will error because `false` (the build_cmd) exits non-zero,
	// but it must NOT error with "no peers configured".
	if err != nil && err.Error() == "no peers configured: pass --connect <addr,...> or add default_peers to the profile" {
		t.Fatal("should not get 'no peers' error when profile has default_peers")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")

	if fileExists(file) {
		t.Error("fileExists(nonexistent) = true, want false")
	}
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileExists(file) {
		t.Error("fileExists(existing file) = false, want true")
	}
	// A directory is not a file.
	if fileExists(dir) {
		t.Error("fileExists(dir) = true, want false")
	}
}
