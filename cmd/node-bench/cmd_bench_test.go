package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBenchRequiresPeers(t *testing.T) {
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
	root.SetArgs([]string{"bench", dir, "HEAD~1", "HEAD"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no peers configured, got nil")
	}
}

func TestBenchRequiresThreeArgs(t *testing.T) {
	root := buildRootCmd()
	root.SetArgs([]string{"bench", "/some/repo"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for wrong arg count, got nil")
	}
}

func TestGitHeadRef(t *testing.T) {
	dir := t.TempDir()
	mustGit(t, dir, "init", "-b", "main")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-m", "init")

	ref, err := gitHeadRef(dir)
	if err != nil {
		t.Fatalf("gitHeadRef: %v", err)
	}
	if ref != "main" {
		t.Errorf("gitHeadRef = %q, want \"main\"", ref)
	}
}

func TestGitHeadRefDetachedHead(t *testing.T) {
	// Create a minimal git repo, detach HEAD, and verify gitHeadRef returns a SHA.
	dir := t.TempDir()
	mustGit(t, dir, "init")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-m", "init")

	sha := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "--short", "HEAD"))
	mustGit(t, dir, "checkout", "--detach", "HEAD")

	ref, err := gitHeadRef(dir)
	if err != nil {
		t.Fatalf("gitHeadRef in detached HEAD: %v", err)
	}
	if ref != sha {
		t.Errorf("gitHeadRef detached = %q, want short SHA %q", ref, sha)
	}
}

func TestGitCheckout(t *testing.T) {
	dir := t.TempDir()
	mustGit(t, dir, "init")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-m", "v1")
	shaV1 := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "--short", "HEAD"))

	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-m", "v2")

	// Switch back to the first commit.
	if err := gitCheckout(dir, shaV1); err != nil {
		t.Fatalf("gitCheckout: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "f"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "v1" {
		t.Errorf("after checkout: file = %q, want \"v1\"", content)
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}
