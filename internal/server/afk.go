// afk.go implements auto-AFK detection and AFK kick.
//
// On each server tick, checkAFK iterates all online players:
//   - If the player hasn't moved for AutoAfkTime, mark them AFK automatically.
//   - If the player is AFK and becomes active again, clear AFK automatically.
//   - If the player is AFK for longer than KickTime (and their rank is below
//     KickMaxRank), kick them with a message.
//
// Movement is tracked via lastAction timestamp, updated on every position
// packet and block change. The check runs every tick (20 TPS) but the
// per-player work is O(1) — just a time comparison.

package server

import "time"

// checkAFK runs the auto-AFK and AFK kick logic on every server tick.
func (s *Server) checkAFK() {
	cfg := s.cfg.AFK
	if cfg.AutoAfkTime <= 0 && cfg.KickTime <= 0 {
		return
	}

	now := time.Now()
	for _, p := range s.codec.OnlinePlayers() {
		la, afkSince, afk := s.codec.GetPlayerAFKState(p.Name())
		if la.IsZero() {
			continue
		}

		if !afk && cfg.AutoAfkTime > 0 {
			// Auto-AFK: mark AFK if inactive for AutoAfkTime.
			if now.Sub(la) >= cfg.AutoAfkTime {
				p.SetAfk(true)
				s.codec.BroadcastMessage("&7" + p.Name() + " is now AFK (auto)")
			}
			continue
		}

		if afk && cfg.AutoAfkTime > 0 {
			// Auto-un-AFK: player became active again.
			if la.After(afkSince) && now.Sub(la) < cfg.AutoAfkTime {
				p.SetAfk(false)
				s.codec.BroadcastMessage("&7" + p.Name() + " is no longer AFK")
				continue
			}
		}

		if afk && cfg.KickTime > 0 {
			// AFK kick: kick if AFK for longer than KickTime.
			// But only if the player's rank is below KickMaxRank.
			rank := s.codec.GetPlayerRank(p.Name())
			if rank >= cfg.KickMaxRank {
				continue
			}
			if !afkSince.IsZero() && now.Sub(afkSince) >= cfg.KickTime {
				p.Kick("Auto-kick: AFK for too long")
			}
		}
	}
}
