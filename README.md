# Solar

**English** | [Русский](README_RU.md)

Solar is an open-source Minecraft Classic and ClassiCube server written in Go.
It provides a small, testable server core with Classic Protocol Extension (CPE)
support, multiple worlds, block physics, moderation, and two plugin options.

> [!WARNING]
> Solar is pre-alpha software. Configuration, storage formats, commands, and
> plugin APIs may change without backward compatibility. Back up your data
> before upgrading and test changes before using them on a public server.

## Why Solar?

- **Classic and ClassiCube focused.** Implements the Classic protocol and a
  broad set of CPE extensions instead of targeting modern Java or Bedrock.
- **Built for extension.** Use the Go plugin API or optionally build with Lua
  scripting support.
- **MCGalaxy-inspired gameplay.** Includes familiar commands, ranks, message
  blocks, portals, doors, and block physics.
- **Operational safeguards.** Provides atomic world saves, configurable TCP
  deadlines, backpressure, authentication, anti-spam controls, structured
  logging, graceful shutdown, and a race-tested codebase.
- **Useful development tools.** Ships with synthetic Classic clients and
  repeatable load-test scenarios.

## What Solar Is Not

Solar is not a vanilla survival server and does not implement modern Minecraft
Java Edition or Bedrock Edition protocols. It has no vanilla mobs, inventory,
crafting, redstone, or survival progression. Use Solar when you want a
Minecraft Classic or ClassiCube building server with custom gameplay.

Solar is an independent project. It is not affiliated with or approved by
Mojang Studios, Microsoft, ClassiCube, or MCGalaxy.

## Features

- Classic client login, world streaming, movement, chat, and entity updates
- CPE negotiation, including FastMap, extended player lists, custom blocks,
  environment controls, selections, plugin messages, and extended teleports
- Multiple loadable worlds with binary `.swld` persistence and automatic saves
- Simple, Classic, fCraft, and heightmap generator families
- Per-level water, lava, sand, gravel, grass, leaves, fire, TNT, and door physics
- Ranks, permissions, whitelist, bans, mute, freeze, hide, AFK, and anti-spam
- Drawing, clipboard, undo/redo, BlockDB history, message blocks, and portals
- English and Russian server messages with per-player language selection
- Go plugins on supported platforms and optional Lua plugins
- Built-in `idle`, `chat`, `move`, `blocks`, and `mixed` load-test scenarios

The canonical command inventory is maintained in
[`docs/COMMAND_MATRIX.md`](docs/COMMAND_MATRIX.md). Release-readiness checks are
tracked in [`docs/GAMEPLAY_MATRIX.md`](docs/GAMEPLAY_MATRIX.md).

## Requirements

- Go 1.24 or newer
- Git and a supported operating system for building from source
- A Minecraft Classic or ClassiCube-compatible client

Go `.so` plugins are loaded only on Linux and macOS. They must be built with
the same Go toolchain and compatible dependency versions as the server.

## Quick Start

```sh
git clone https://github.com/solar-mc/solar.git
cd solar
make build
./bin/solar start --config configs/server.toml
```

You can also run directly from source:

```sh
go run ./cmd/solar start --config configs/server.toml
```

Connect your client to `127.0.0.1:25565`. On first start, Solar creates its
data directories and generates the configured main world when no saved world
exists.

## Configuration

The example configuration is [`configs/server.toml`](configs/server.toml).
It documents every supported setting and is the recommended starting point.

```toml
listen = ":25565"
data_dir = "data"
max_players = 128
autosave_interval = "60s"
default_generator = "Classic"
server_name = "Solar"

[storage]
main_world_name = "main"

[language]
default = "en"
file = "configs/language.toml"

[auth]
enabled = false
salt = ""

[heartbeat]
enabled = false
public = false
```

Important operational settings:

| Area | Settings |
| --- | --- |
| Network | `listen`, `max_players`, `[network]` timeouts and queue limits |
| Worlds | `default_generator`, `[world]`, `[storage]`, `autosave_interval` |
| Access | `operators`, `[auth]`, `[heartbeat]`, `[player]` whitelist |
| Abuse controls | `[antispam]` and `[afk]` |
| Extensions | `[cpe]`, `[plugins]`, and `[lua]` |
| Operations | `[log]` and `[debug].pprof_address` |

Operators may also be seeded with `SOLAR_OPERATORS="alice,bob"`.

### Public Servers

Before publishing a server:

1. Enable `[auth]` so Classic `mppass` usernames are verified.
2. Enable and configure `[heartbeat]` if the server should appear on the
   ClassiCube server list.
3. Make the configured TCP port reachable from the internet.
4. Keep `data/` on persistent storage and back it up regularly.
5. Run the production gate and test with a real ClassiCube client.

## Server Commands

Run `/help` in game to list commands available to your rank. Major command
groups include:

- World management: `/newlvl`, `/load`, `/unload`, `/goto`, `/save`, `/map`
- Moderation: `/kick`, `/ban`, `/whitelist`, `/mute`, `/freeze`, `/hide`
- Teleportation: `/tp`, `/spawn`, `/back`, `/tpa`, `/tpaccept`, `/tpdeny`
- Building: `/cuboid`, `/line`, `/sphere`, `/fill`, `/copy`, `/paste`
- Interactive blocks: `/mb`, `/portal`, `/door`
- History and recovery: `/blockdb`, `/blockundo`, `/undo`, `/redo`
- Ranks and information: `/setrank`, `/rankinfo`, `/viewranks`, `/whois`

See the [command matrix](docs/COMMAND_MATRIX.md) for the canonical list and
permission level of each command.

## Plugins

### Go Plugins

The public API is under [`plugin/`](plugin/). A minimal dynamically loaded
plugin is available at [`plugins/soplug`](plugins/soplug).

Plugin commands can declare aliases, help text, and a minimum rank through
`plugin.CommandSpec`. The player API exposes rank/build permissions and
interactive block marking through `SelectBlocks`; persistent global plugin
files belong in the directory returned by `Server.PluginDataDir`.

```sh
go build -tags=plugin -buildmode=plugin \
  -o data/plugins/soplug.so ./plugins/soplug
```

Then enable `[plugins]` in `configs/server.toml`. Rebuild plugins whenever the
server's Go version or dependencies change.

The [`plugins/mcgalaxy`](plugins/mcgalaxy) command packs add MCGalaxy-style
commands without adding them to the server core. Each command category is an
independent plugin; all three can be loaded together:

```sh
go build -tags=plugin -buildmode=plugin \
  -o data/plugins/mcgalaxy-chat.so ./plugins/mcgalaxy/chat
go build -tags=plugin -buildmode=plugin \
  -o data/plugins/mcgalaxy-cpe.so ./plugins/mcgalaxy/cpe
go build -tags=plugin -buildmode=plugin \
  -o data/plugins/mcgalaxy-economy.so ./plugins/mcgalaxy/economy
```

The chat plugin provides `/roll`, `/8ball`, `/high5`, `/hug`, `/eat`, `/say`,
`/clear`, and `/color`. The CPE plugin provides `/reachdistance`. The economy
plugin provides `/balance`, `/pay`, `/give`, `/take`, and `/economy`, with
state stored atomically in its private data directory.

### Lua Plugins

Lua support is excluded from the default binary. Build it explicitly:

```sh
go build -tags=lua -o bin/solar ./cmd/solar
mkdir -p data/lua
cp plugins/example.lua data/lua/example.lua
```

Enable `[lua]` in the configuration before starting the server. The example at
[`plugins/example.lua`](plugins/example.lua) demonstrates events, commands,
player operations, levels, and scheduled tasks.

## Load Testing

Solar includes a synthetic Classic client runner:

```sh
./bin/solar loadtest \
  --address 127.0.0.1:25565 \
  --clients 64 \
  --duration 30s \
  --scenario mixed \
  --cpe
```

When authentication is enabled, add `--auth-salt <salt>`.

## Development

```sh
make test          # unit and integration tests
make test-race     # tests with the Go race detector
make vet           # static checks
make lint          # golangci-lint
make build         # build bin/solar
make production-gate
```

The production gate runs tests, `go vet`, race detection, a mixed CPE load
test, health checks, and optionally a real ClassiCube smoke test.

Repository map:

| Path | Purpose |
| --- | --- |
| `cmd/solar/` | CLI entry point |
| `internal/protocol/classic/` | Classic and CPE protocol implementation |
| `internal/server/` | Server lifecycle, levels, physics, and persistence |
| `internal/command/` | Command registry and built-in commands |
| `internal/blocks/` | Block definitions, physics, history, and drawing logic |
| `internal/generator/` | Built-in world generators |
| `plugin/` | Public Go plugin API |
| `internal/loadtest/` | Synthetic Classic clients |
| `third_party/` | Upstream reference source and assets |

Contributions are welcome. Read [`CONTRIBUTING.md`](CONTRIBUTING.md), keep
changes focused, and add tests for behavior changes.

## License

Solar is released under the [MIT License](LICENSE).

Third-party source and assets under `third_party/` retain their respective
upstream licenses.
