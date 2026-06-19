# Solar

Solar is a Go implementation of a Minecraft Classic/ClassiCube-compatible server. The project focuses on a small, testable core: Classic protocol handling, world persistence, player policy, chat commands, and load testing tools.

## Status

Solar is pre-alpha. The server can accept Classic clients, stream a world, track players/entities, process basic commands, persist world state, and run synthetic load tests.

## Quick Start

```sh
go test ./internal/...
go run ./cmd/solar start --config configs/server.toml
```

Connect a Classic/ClassiCube client to `127.0.0.1:25565`.

## Configuration

Solar reads `configs/server.toml` by default:

```toml
listen = ":25565"
data_dir = "data"
workers = 8
max_players = 128
connect_rate = 32
server_name = "Solar"
motd = "CLI-only classic server"
operators = ["alice", "bob"]
autosave_interval = "60s"
```

Operators can also be seeded with `SOLAR_OPERATORS="alice,bob"` or the legacy `SOLAR_ADMIN` variable.

## Commands

- `/help` lists available commands.
- `/where` shows your current position.
- `/setblock x y z block` changes a block.
- `/tp x y z [yaw pitch]` teleports yourself.
- `/setspawn [x y z [yaw pitch]]` updates world spawn.
- `/save` queues a world and policy save.
- `/kick`, `/ban`, `/unban`, `/whitelist`, and `/players` manage moderation.

## Load Testing

```sh
go run ./cmd/solar loadtest --address 127.0.0.1:25565 --clients 64 --duration 30s --scenario mixed --cpe
```

Supported scenarios: `idle`, `chat`, `move`, `blocks`, `mixed`.

## Project Layout

```text
cmd/solar/               CLI entrypoint
internal/app/            command runners and bootstrap wiring
internal/cli/            CLI parsing
internal/config/         config loading, defaults, validation (TOML)
internal/server/         TCP server lifecycle, graceful shutdown, persistence
internal/session/        online room/session fan-out primitives
internal/protocol/classic/ Classic/ClassiCube wire protocol
internal/world/          world model, persistence, generation, level mapping
internal/player/         player registry and moderation policy (bans/whitelist/ops)
internal/entity/         entity state and simulation
internal/command/        command registry and built-ins
internal/generator/      map generators (Simple, Classic, fCraft, Heightmap)
internal/loadtest/       synthetic Classic clients
internal/storage/        local file-backed storage paths
internal/worker/         fixed-size background job pool
third_party/             upstream reference projects and assets
```

## Development

```sh
make fmt        # format all Go source
make vet        # go vet
make test       # run tests
make test-race  # run tests with race detector
make lint       # golangci-lint
make ci         # vet + race tests + lint + build
make build      # build the solar binary
make docker     # build the Docker image
```

Keep protocol code deterministic and covered with wire-format tests. Avoid introducing runtime dependencies unless they clearly improve operational quality. See [CONTRIBUTING.md](CONTRIBUTING.md) for style guidelines.
