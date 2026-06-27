# Gameplay/Core Matrix

This is the minimum matrix for production readiness. Each row needs either a real ClassiCube check or a protocol-level loadtest check before release.

| Area | Checks | Gate |
| --- | --- | --- |
| Authentication | valid `mppass`, invalid `mppass`, duplicate names | `scripts/real-classicube-smoke.sh` |
| Chat | normal chat, long chat, color codes, command parsing, spam limits | protocol scenario + manual two-client check |
| Ranks | guest/member/op/admin permissions, deny-by-default | command conformance tests + real client |
| Moderation | kick, ban, unban, mute, whitelist | command conformance tests + real client |
| Movement | spawn, teleport, invalid coordinates, AFK | protocol scenario + real client |
| Blocks | place, break, invalid coords, permissions, undo | mixed loadtest + command tests |
| Physics | sand/water/lava/special block side effects | block physics tests + real client spot check |
| World | generate, load, save, autosave, restart | persistence tests + shutdown smoke |
| CPE | negotiation, FastMap, block definitions, player list | mixed CPE loadtest |
| Backpressure | slow clients, full outbox, disconnect behavior | loadtest + race gate |
| Plugins | event panic isolation, Lua boundaries, scheduler cleanup | plugin tests |
| Storage | path traversal rejection, atomic writes, crash-safe restore | storage tests + backup restore smoke |

Production means every touched row stays green in the release gate or has a written exception here.
See `docs/COMMAND_MATRIX.md` for the full command list.
