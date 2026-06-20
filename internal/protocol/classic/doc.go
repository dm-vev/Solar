// Package classic implements the Minecraft Classic / ClassiCube
// wire protocol for Solar.
//
// The package is organized by responsibility:
//
//   - [Codec] manages the protocol codec lifecycle, TCP session creation,
//     and configuration (timeouts, outbox, block definitions, i18n).
//
//   - [Session] holds per-connection state: identity, entity tracking,
//     read/write loops, and the plugin.Player interface implementation.
//
//   - [SessionBackend] is the interface that command handlers use to
//     interact with the session (teleport, moderation, level ops, etc.).
//
//   - handlers.go processes incoming packets (set block, chat, movement).
//
//   - conn.go implements the asynchronous write loop and connection
//     lifecycle (packet queue, flush, disconnect).
//
//   - encode.go contains pure wire-format encoding functions.
//
//   - broadcast.go handles entity visibility, room join/leave, and
//     per-level broadcast filtering.
//
//   - handshake.go implements the login handshake, CPE negotiation,
//     level streaming, and map switching.
//
//   - cpe.go and cpe_encode.go implement ClassiCube Protocol Extension
//     negotiation and packet encoding.
//
//   - cpe_handlers.go processes CPE-specific incoming packets.
//
//   - cpe_plugin.go implements the plugin.CPE interface on Session.
//
//   - env.go sends CPE environment packets based on level.Env settings.
//
//   - admin.go implements all SessionBackend methods (moderation,
//     level management, block DB, info services).
//
//   - plugin_player.go implements the plugin.Player interface on Session.
//
//   - state.go provides safe state accessors (identity, flags, CPE support).
//
//   - opcodes.go defines the Classic protocol packet IDs.
//
//   - blocks.go sends custom block definitions to clients.
//
//   - level_stream.go encodes the level data stream (gzip or fastmap).
package classic
