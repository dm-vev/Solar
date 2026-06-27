# Solar Production Gate

Run this before calling the core production-ready:

```bash
scripts/production-gate.sh
```

The gate runs:

- `go test ./...`.
- `go vet ./...`.
- `go test -race ./...`.
- 254 mixed CPE protocol clients with auth enabled.
- Two real ClassiCube clients with valid `mppass` and one rejected invalid client under isolated `Xvfb`.

Use `SOLAR_SKIP_REAL_CLIENT=1` only on machines without `Xvfb` or the vendored ClassiCube binary.
Use `SOLAR_LOAD_DURATION=1h` for a soak run.

## Runtime Baseline

- Keep `[auth].enabled = true` on public servers.
- Keep `[heartbeat].enabled = true` for public ClassiCube listing.
- Set explicit `[auth].salt` when heartbeat is disabled.
- Keep `max_players <= 254` for Classic protocol compatibility unless the client/protocol path changes.
- Use `autosave_interval` in production; `0s` is only for tests and gates.
- Keep `pprof_address` bound to localhost or disabled.

## Backup And Restore

- Back up `data/worlds`, `data/players`, `data/blockdb`, and config together.
- Stop Solar or run a save before filesystem snapshots.
- Restore into a fresh `data_dir`; do not merge partial world/player files.

## Release Checklist

- `scripts/production-gate.sh` passes.
- `SOLAR_LOAD_DURATION=1h scripts/production-gate.sh` passes for large-server releases.
- Real ClassiCube smoke artifacts show `players=2` and invalid auth stays rejected.
- No uncommitted changes except release metadata.
