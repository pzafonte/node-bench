package nodeprofile

import (
	"os"
	"testing"
)

func TestResolveBareNameErrors(t *testing.T) {
	// Bare names (no path indicators) are no longer valid — there are no built-ins.
	for _, name := range []string{"kernel-node", "btcd", "bitcoin-core"} {
		_, err := Resolve(name)
		if err == nil {
			t.Errorf("Resolve(%q): expected error, got nil", name)
		}
	}
}

func TestResolveDirectoryMissingToml(t *testing.T) {
	dir := t.TempDir()
	_, err := Resolve(dir)
	if err == nil {
		t.Fatal("expected error for directory without .node-bench.toml, got nil")
	}
}

func TestLoadFromFile(t *testing.T) {
	p, err := LoadFromFile("example.node-bench.toml")
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if p.Name != "my-node" {
		t.Errorf("Name = %q, want %q", p.Name, "my-node")
	}
	if p.BinaryPath != "target/release/my-node" {
		t.Errorf("BinaryPath = %q, want %q", p.BinaryPath, "target/release/my-node")
	}
	if p.Logs.UpdateTip != "UpdateTip" {
		t.Errorf("Logs.UpdateTip = %q, want %q", p.Logs.UpdateTip, "UpdateTip")
	}
	t.Logf("loaded: name=%s build=%v binary=%s", p.Name, p.BuildCmd, p.BinaryPath)
}

func TestResolveFilePath(t *testing.T) {
	p, err := Resolve("example.node-bench.toml")
	if err != nil {
		t.Fatalf("Resolve(file path): %v", err)
	}
	if p.Name != "my-node" {
		t.Errorf("Name = %q, want %q", p.Name, "my-node")
	}
}

func TestResolveDirectory(t *testing.T) {
	dir := t.TempDir()
	content := `
name        = "dir-node"
build_cmd   = ["go", "build", "."]
binary_path = "mynode"
[logs]
update_tip = "NewTip"
`
	if err := writeFile(dir+"/.node-bench.toml", content); err != nil {
		t.Fatal(err)
	}
	p, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve(dir): %v", err)
	}
	if p.Name != "dir-node" {
		t.Errorf("Name = %q, want %q", p.Name, "dir-node")
	}
}

func TestLoadFromFileDefaultPeers(t *testing.T) {
	withPeers := t.TempDir() + "/with-peers.toml"
	if err := writeFile(withPeers, `
name        = "peer-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/peer-node"
default_peers = ["1.2.3.4:38333", "5.6.7.8:38333"]
`); err != nil {
		t.Fatal(err)
	}
	p, err := LoadFromFile(withPeers)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if len(p.DefaultPeers) != 2 {
		t.Fatalf("DefaultPeers len = %d, want 2", len(p.DefaultPeers))
	}
	if p.DefaultPeers[0] != "1.2.3.4:38333" {
		t.Errorf("DefaultPeers[0] = %q, want \"1.2.3.4:38333\"", p.DefaultPeers[0])
	}
	if p.DefaultPeers[1] != "5.6.7.8:38333" {
		t.Errorf("DefaultPeers[1] = %q, want \"5.6.7.8:38333\"", p.DefaultPeers[1])
	}

	// Profile without default_peers --> field is nil (not an empty non-nil slice).
	noPeers := t.TempDir() + "/no-peers.toml"
	if err := writeFile(noPeers, `
name        = "no-peers-node"
build_cmd   = ["echo", "build"]
binary_path = "bin/no-peers-node"
`); err != nil {
		t.Fatal(err)
	}
	p2, err := LoadFromFile(noPeers)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if p2.DefaultPeers != nil {
		t.Errorf("DefaultPeers = %v, want nil when field is absent", p2.DefaultPeers)
	}
}

func TestLoadFromFileMissingFields(t *testing.T) {
	f := t.TempDir() + "/bad.toml"
	if err := writeFile(f, `name = ""`); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFromFile(f)
	if err == nil {
		t.Fatal("expected error for missing fields, got nil")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
