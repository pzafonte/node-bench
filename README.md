# node-bench

`node-bench` is a node-agnostic high-level performance benchmark harness. Right now, it measures Initial Block Download (IBD) throughput, across commits, giving you a longitudinal record of sync performance.

Complementary to microbenchmark tools: those tell you *which function* regressed and by how much in isolation; node-bench tells you *whether it matters* at a high level.

## Supported nodes

| Node | Status |
|------|--------|
| [kernel-node](https://github.com/kernel-node/kernel-node) | primary target |
| [btcd](https://github.com/btcsuite/btcd) | supported |

## Requirements

- Go 1.21+
- Git
- A Bitcoin node repo with a `.node-bench.toml` profile (or a standalone `.toml` file)
- A synced peer on the target network (signet, testnet, mainnet)

## Install

```sh
go install github.com/pzafonte/node-bench/cmd/node-bench@latest
```

Or build from source:

```sh
git clone https://github.com/pzafonte/node-bench
cd node-bench
go build -o node-bench ./cmd/node-bench
```

## Profile file (`.node-bench.toml`)

Place a `.node-bench.toml` at the root of the node repo you want to benchmark, or pass a path with `--profile`.

```toml
name        = "my-node"
build_cmd   = ["cargo", "build", "--release"]
binary_path = "target/release/my-node"
extra_args  = ["--nolisten"]
default_peers = ["1.2.3.4:38333"]

[flags]
network_flag = "--signet"   # boolean flag — appended as-is
datadir      = "--datadir"
connect      = "--connect"

[logs]
update_tip = "UpdateTip:"  # log line pattern that marks a new block
```

See `profiles/` for ready-made examples.

## Usage

### `run` — single IBD benchmark

```sh
node-bench run <repo-path> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--profile` | `.node-bench.toml` in repo | Path to profile file |
| `--connect` | profile `default_peers` | Peer address(es) to sync from |
| `--datadir` | temp dir, cleaned after | Data directory for the node |
| `--network` | `signet` | Network to benchmark on |
| `--duration` | `5m` | How long to run the node before stopping |
| `--trials` | `1` | Number of independent runs |

Result is saved to `results/<commit>.json`.

### `bench` — compare two commits

```sh
node-bench bench <repo-path> <ref-A> <ref-B> [flags]
```

Checks out ref-A, runs `run`, then checks out ref-B, runs `run`, and prints a side-by-side comparison. Restores HEAD when done.

### `profile` — inspect a profile

```sh
node-bench profile <repo-path-or-toml>
```

Prints a human-readable summary of the resolved profile. Useful for debugging before a benchmark run.

### `history` — view past results

```sh
node-bench history [results-dir]
```

Prints all saved results sorted by most-recent first. Defaults to `./results`.

### `compare` — compare two saved results

```sh
node-bench compare <result-A.json> <result-B.json>
```

Prints a delta table: absolute and percentage change for header sync, block sync, BPS, and height.

## Results

Results are stored as JSON in `results/<short-sha>.json`:

```json
{
  "commit": "9c6fade0",
  "branch": "feature/psbtv2-bip375",
  "network": "signet",
  "blocks_per_sec": 384,
  "max_height": 112122
}
```

Commit the `results/` directory to build a longitudinal performance record.
