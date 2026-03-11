package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func runProfileCmd(t *testing.T, repoPath string) string {
	t.Helper()
	cmd := cmdProfile()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	// Cobra needs a root to call Execute; use a throwaway parent.
	root := &cobra.Command{Use: "test"}
	root.AddCommand(cmd)
	root.SetArgs([]string{"profile", repoPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("profile command: %v", err)
	}
	return buf.String()
}

func TestProfileCmdNoToml(t *testing.T) {
	dir := t.TempDir()
	cmd := cmdProfile()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	root := &cobra.Command{Use: "test"}
	root.AddCommand(cmd)
	root.SetArgs([]string{"profile", dir})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error for dir without .node-bench.toml, got nil; output:\n%s", buf.String())
	}
}

func TestProfileCmdFromToml(t *testing.T) {
	dir := t.TempDir()
	toml := `
name        = "toml-node"
build_cmd   = ["cargo", "build"]
binary_path = "target/release/toml-node"
[logs]
update_tip = "Tip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runProfileCmd(t, dir)
	if !strings.Contains(out, "name:         toml-node") {
		t.Errorf("expected toml-node in output, got:\n%s", out)
	}
	if !strings.Contains(out, ".node-bench.toml") {
		t.Errorf("expected .node-bench.toml in source line, got:\n%s", out)
	}
}

func TestProfileCmdShowsDefaultPeers(t *testing.T) {
	dir := t.TempDir()
	toml := `
name        = "peer-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/peer-node"
default_peers = ["1.2.3.4:38333", "5.6.7.8:38333"]
[logs]
update_tip = "Tip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runProfileCmd(t, dir)
	if !strings.Contains(out, "1.2.3.4:38333") {
		t.Errorf("expected peer 1.2.3.4:38333 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "5.6.7.8:38333") {
		t.Errorf("expected peer 5.6.7.8:38333 in output, got:\n%s", out)
	}
}

func TestProfileCmdNetworkFlag(t *testing.T) {
	// network_flag (boolean-style) must appear in output; Network must not.
	dir := t.TempDir()
	toml := `
name        = "signet-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/signet-node"
[flags]
network_flag = "--signet"
datadir      = "--datadir"
connect      = "--connect"
[logs]
update_tip = "Tip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runProfileCmd(t, dir)
	if !strings.Contains(out, "--signet") {
		t.Errorf("expected --signet in output, got:\n%s", out)
	}
	if !strings.Contains(out, "boolean flag") {
		t.Errorf("expected 'boolean flag' annotation for network_flag, got:\n%s", out)
	}
}

func TestProfileCmdExtraArgs(t *testing.T) {
	dir := t.TempDir()
	toml := `
name        = "extra-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/extra-node"
extra_args  = ["--nolisten", "--maxpeers=1"]
[logs]
update_tip = "Tip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runProfileCmd(t, dir)
	if !strings.Contains(out, "--nolisten") {
		t.Errorf("expected --nolisten in output, got:\n%s", out)
	}
	if !strings.Contains(out, "--maxpeers=1") {
		t.Errorf("expected --maxpeers=1 in output, got:\n%s", out)
	}
}

func TestProfileCmdNoExtraArgs(t *testing.T) {
	// extra_args line must not appear when the field is absent.
	dir := t.TempDir()
	toml := `
name        = "plain-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/plain-node"
[logs]
update_tip = "Tip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runProfileCmd(t, dir)
	if strings.Contains(out, "extra_args") {
		t.Errorf("extra_args line should not appear when field is absent, got:\n%s", out)
	}
}

func TestProfileCmdNoPeersMessage(t *testing.T) {
	dir := t.TempDir()
	toml := `
name        = "no-peers-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/no-peers-node"
[logs]
update_tip = "Tip"
`
	if err := os.WriteFile(filepath.Join(dir, ".node-bench.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runProfileCmd(t, dir)
	if !strings.Contains(out, "--connect required") {
		t.Errorf("expected '--connect required' message for profile without peers, got:\n%s", out)
	}
}
