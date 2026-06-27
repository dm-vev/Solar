# Command Matrix

Source of truth: `internal/command/registry.go`.

| Command | Rank | Area | Gate |
| --- | --- | --- | --- |
| `/help` | guest | info | command tests, real client spot check |
| `/where` | guest | info | command tests |
| `/setblock` | operator | world edit | command tests, mixed loadtest |
| `/tp` | operator | teleport | command tests, real client spot check |
| `/setspawn` | operator | world | command tests |
| `/setspawnpoint` | operator | world | command tests |
| `/save` | operator | persistence | command tests, shutdown smoke |
| `/kick` | operator | moderation | command tests, real client spot check |
| `/ban` | operator | moderation | command tests |
| `/unban` | operator | moderation | command tests |
| `/whitelist` | admin command | moderation | command tests |
| `/players` | guest | info | command tests |
| `/newlvl` | operator | world | command tests |
| `/gb` | admin command | block definitions | command tests |
| `/lb` | admin command | block definitions | command tests |
| `/blockdb` | operator | audit/history | command tests |
| `/about` | guest | blockdb | command tests |
| `/b` | guest | blockdb alias | command tests |
| `/blockundo` | guest | blockdb | command tests |
| `/load` | operator | levels | command tests |
| `/unload` | operator | levels | command tests |
| `/reload` | operator | levels | command tests |
| `/levels` | guest | levels | command tests |
| `/goto` | guest | levels | command tests, real client spot check |
| `/main` | guest | levels | command tests |
| `/physics` | operator | physics | command tests |
| `/map` | operator | map env | command tests |
| `/mute` | operator | moderation | command tests, chat smoke |
| `/unmute` | operator | moderation | command tests |
| `/freeze` | operator | moderation | command tests |
| `/unfreeze` | operator | moderation | command tests |
| `/summon` | operator | teleport | command tests |
| `/afk` | guest | player state | command tests |
| `/hide` | guest | player state | command tests |
| `/seen` | guest | info | command tests |
| `/whois` | guest | info | command tests |
| `/blocks` | guest | info | command tests |
| `/mapinfo` | guest | info | command tests |
| `/serverinfo` | guest | info | command tests |
| `/time` | guest | info | command tests |
| `/rules` | guest | info | command tests |
| `/spawn` | guest | teleport | command tests, real client spot check |
| `/back` | guest | teleport | command tests |
| `/tpa` | guest | teleport | command tests |
| `/me` | guest | chat | command tests, chat smoke |
| `/whisper` | guest | chat | command tests, two-client real check |
| `/ignore` | guest | chat | command tests |
| `/cuboid` | guest | drawing | command tests |
| `/line` | guest | drawing | command tests |
| `/sphere` | guest | drawing | command tests |
| `/fill` | guest | drawing | command tests |
| `/copy` | guest | clipboard | command tests |
| `/paste` | guest | clipboard | command tests |
| `/mb` | guest | special blocks | command tests |
| `/portal` | guest | special blocks | command tests |
| `/door` | guest | special blocks | command tests |
| `/undo` | guest | undo | command tests |
| `/redo` | guest | undo | command tests |
| `/setrank` | operator | ranks | command tests |
| `/rankinfo` | guest | ranks | command tests |
| `/viewranks` | guest | ranks | command tests |
