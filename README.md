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

Solar reads `configs/server.toml` by default. Top-level keys control core
server parameters; nested `[table]` sections tune network, simulation, world,
debug, and logging:

```toml
listen = ":25565"
data_dir = "data"
workers = 8
max_players = 128
connect_rate = 32
autosave_interval = "60s"
default_generator = "Classic"
server_name = "Solar"
motd = "CLI-only classic server"
operators = ["alice", "bob"]

[network]
read_timeout = "30s"
write_timeout = "10s"
tcp_nodelay = true
session_outbox_size = 256
write_batch_size = 32

[simulation]
tick_interval = "50ms"

[world]
default_width = 128
default_height = 64
default_length = 128
max_blocks = 67108864

[storage]
backend = "local"
worlds_dir = "worlds"
players_dir = "players"
policy_file = "policy.json"
world_file_ext = ".swld"
main_world_name = "main"

[commands]
admin_commands = ["tp", "setspawn", "save", "kick", "ban", "unban", "whitelist", "newlvl"]

[player]
whitelist_enabled = false
max_username_length = 32

[cpe]
ext_player_list = true
fast_map = true
two_way_ping = true

[debug]
pprof_address = ""
pprof_shutdown_timeout = "5s"

[log]
level = "info"
format = "text"
```

`autosave_interval` accepts a duration; use `0s` to disable automatic saves.
Negative values are rejected.

Operators can also be seeded with `SOLAR_OPERATORS="alice,bob"` or the legacy
`SOLAR_ADMIN` variable.

`[network]` — `read_timeout` / `write_timeout` set per-session TCP deadlines
(0 disables); `session_outbox_size` is the per-client packet queue depth
(client is disconnected when full); `write_batch_size` controls how many
packets are coalesced per flush.

`[simulation]` — `tick_interval` drives world and entity updates (50ms = 20 TPS).

`[world]` — `default_*` dimensions are used when generating a fresh world;
`max_blocks` is the hard volume cap (default 64M).

`[storage]` — `backend` selects the persistence backend (only `"local"`);
`worlds_dir`/`players_dir`/`policy_file`/`world_file_ext`/`main_world_name`
control the on-disk layout.

`[commands]` — `admin_commands` lists commands that require operator privileges.

`[player]` — `whitelist_enabled` starts the server with whitelist enforcement
on; `max_username_length` caps accepted usernames (protocol max 64).

`[cpe]` — toggle individual CPE extensions the server advertises to clients.

`[debug]` — `pprof_address` enables the pprof/health HTTP server
(`--pprof` CLI flag overrides); `pprof_shutdown_timeout` controls graceful
shutdown of that server.

`[log]` — `level` is one of `debug|info|warn|error`; `format` is `text` or `json`.

## Commands

- `/help` lists available commands.
- `/where` shows your current position.
- `/setblock x y z block` changes a block.
- `/tp x y z [yaw pitch]` teleports yourself.
- `/setspawn [x y z [yaw pitch]]` updates world spawn.
- `/save` queues a world and policy save.
- `/kick`, `/ban`, `/unban`, `/whitelist`, and `/players` manage moderation.
- `/gb` and `/lb` manage custom block definitions (see Custom Blocks below).

## Custom Blocks

Solar supports custom block definitions via the ClassiCube CPE
`BlockDefinitions` and `BlockDefinitionsExt` extensions. Definitions
are stored as JSON in `data/blockdefs/`.

### Commands

```
/gb add <id> [name]              — create a custom block (ID 66-255)
/gb edit <id> <property> <value> — modify a block property
/gb remove <id>                  — delete a custom block
/gb info <id>                    — show block properties
/gb list                         — list all custom blocks
```

`/lb` is an alias for `/gb` (both modify the shared registry).

### Editable properties

`name`, `collide` (0-7), `speed`, `toptex`, `sidetex`, `alltex`,
`bottomtex`, `lefttex`, `righttex`, `fronttex`, `backtex`,
`blockslight` (true/false), `sound` (0-9), `fullbright` (true/false),
`shape` (0=sprite, 16=cube), `blockdraw` (0-5), `fallback` (0-65),
`fogdensity`, `fogcolor` (r g b), `min` (x y z), `max` (x y z).

### Persistence

Block definitions are saved to `data/blockdefs/global.json` and loaded
at startup. The file format is a JSON array of block definition objects.

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
internal/blocks/          block definitions, editing history, and per-level physics
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
