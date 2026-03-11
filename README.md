# node-bench

`node-bench` is a node-agnostic high-level performance monitoring harness. Right now, it measures Initial Block Download (IBD) throughput, across commits, giving you a longitudinal record of sync performance.

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

## Future development

### Near-term

- **Prometheus metrics export** — parse structured node metrics (if available) alongside log-based timing, giving finer-grained IBD breakdowns (UTXO cache hit rate, peer stall time, script validation time)
- **CI integration** — `node-bench ci` subcommand that writes a machine-readable summary suitable for GitHub Actions step output and PR comment bots; catches IBD regressions before merge
- **Checkpoint-based progress curves** — plot BPS over time within a single run (not just the aggregate), exposing warm-up effects and stall patterns
- **Mainnet and testnet4 profiles** — signet is fast and cheap; mainnet benchmarks require more infrastructure but produce more realistic numbers

### Medium-term

- **Broader node support** — libbitcoin, bcoin, and any implementation that exposes a parseable log or metrics endpoint; the `.node-bench.toml` adapter pattern is designed for this
- **Hardware normalisation** — record CPU model, core count, RAM, and disk type in `results/` so cross-machine comparisons carry enough context to be meaningful
- **Statistical rigor** — automatic trial count recommendation based on observed variance; flag results where stddev is too high to trust the comparison
- **Snapshot-based IBD** — start from a known UTXO snapshot (BIP 157 / assumeUTXO) to benchmark block validation in isolation, independent of header sync time

### Long-term

- **Distributed benchmark fleet** — run trials across multiple machines simultaneously and aggregate; useful for catching hardware-specific regressions
- **Block validation profiling** — integrate with `pprof` or `perf` to attach a CPU profile to each `results/` entry, bridging the gap between macrobenchmark signal and microbenchmark root cause

## License

MIT
