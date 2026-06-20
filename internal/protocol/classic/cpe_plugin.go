package classic

import "github.com/solar-mc/solar/plugin"

var _ plugin.CPE = (*session)(nil)

// CPE returns the CPE packet sender for this player.
func (s *session) CPE() plugin.CPE { return s }

func (s *session) SetEnvColor(slot byte, r, g, b byte) {
	if !s.supportsExt(cpeExtEnvColors) {
		return
	}
	_ = s.writePacket(encodeEnvColor(slot, int16(r), int16(g), int16(b)))
}

func (s *session) SetWeather(weather byte) {
	if !s.supportsExt(cpeExtEnvWeatherType) {
		return
	}
	_ = s.writePacket(encodeEnvWeatherType(weather))
}

func (s *session) SetHackControl(flying, noclip, speed, respawnHeight, thirdPerson bool) {
	if !s.supportsExt(cpeExtHackControl) {
		return
	}
	// ponytail: maxJumpHeight defaults to 0; add a param if plugins need it.
	_ = s.writePacket(encodeHackControl(flying, noclip, speed, respawnHeight, thirdPerson, 0))
}

func (s *session) SetClickDistance(distance float64) {
	if !s.supportsExt(cpeExtClickDistance) {
		return
	}
	_ = s.writePacket(encodeClickDistance(int16(distance)))
}

func (s *session) SetTextHotkey(key byte, action string, flags byte) {
	if !s.supportsExt(cpeExtTextHotkey) {
		return
	}
	_ = s.writePacket(encodeTextHotkey(action, action, int32(key), flags))
}

func (s *session) HoldThis(block byte, canChange bool) {
	if !s.supportsExt(cpeExtHeldBlock) {
		return
	}
	_ = s.writePacket(encodeHoldThis(block, !canChange))
}

func (s *session) SetBlockPermission(block byte, allowPlace, allowDelete bool) {
	if !s.supportsExt(cpeExtBlockPermissions) {
		return
	}
	_ = s.writePacket(encodeSetBlockPermission(block, allowPlace, allowDelete))
}

func (s *session) ChangeModel(entityID byte, model string) {
	if !s.supportsExt(cpeExtChangeModel) {
		return
	}
	_ = s.writePacket(encodeChangeModel(entityID, model))
}

func (s *session) SetMapAppearance(url string, sideLevel byte) {
	if !s.supportsExt(cpeExtEnvMapAppearance) {
		return
	}
	// ponytail: sideBlock/edgeBlock default to 0; sideLevel maps to edgeLevel.
	_ = s.writePacket(encodeMapAppearance(url, 0, 0, int16(sideLevel)))
}

func (s *session) SetInventoryOrder(block byte, order byte) {
	if !s.supportsExt(cpeExtInventoryOrder) {
		return
	}
	_ = s.writePacket(encodeSetInventoryOrder(block, order))
}

func (s *session) SetHotbar(slot byte, block byte) {
	if !s.supportsExt(cpeExtSetHotbar) {
		return
	}
	_ = s.writePacket(encodeSetHotbar(block, slot))
}

func (s *session) SetSpawnpoint(x, y, z int, yaw, pitch byte) {
	if !s.supportsExt(cpeExtSetSpawnpoint) {
		return
	}
	_ = s.writePacket(encodeSetSpawnpoint(x, y, z, yaw, pitch))
}

func (s *session) SendEffect(effectID, x, y, z byte) {
	if !s.supportsExt(cpeExtCustomParticles) {
		return
	}
	// ponytail: origin set to spawn position; add params if plugins need separate origin.
	xf, yf, zf := float32(x), float32(y), float32(z)
	_ = s.writePacket(encodeSpawnEffect(effectID, xf, yf, zf, xf, yf, zf))
}

func (s *session) SendSelection(id byte, label string, minX, minY, minZ, maxX, maxY, maxZ int, r, g, b byte) {
	if !s.supportsExt(cpeExtSelectionCuboid) {
		return
	}
	// ponytail: opacity defaults to 255 (fully opaque); add a param if plugins need it.
	_ = s.writePacket(encodeMakeSelection(id, label,
		uint16(minX), uint16(minY), uint16(minZ),
		uint16(maxX), uint16(maxY), uint16(maxZ),
		int16(r), int16(g), int16(b), 255))
}

func (s *session) RemoveSelection(id byte) {
	if !s.supportsExt(cpeExtSelectionCuboid) {
		return
	}
	_ = s.writePacket(encodeRemoveSelection(id))
}

func (s *session) SetMapProperty(property byte, value int) {
	if !s.supportsExt(cpeExtEnvMapAspect) {
		return
	}
	_ = s.writePacket(encodeSetMapEnvProperty(property, int32(value)))
}

func (s *session) SetEntityProperty(entityID byte, property byte, value int) {
	if !s.supportsExt(cpeExtEntityProperty) {
		return
	}
	_ = s.writePacket(encodeSetEntityProperty(entityID, property, int32(value)))
}

func (s *session) SendPluginMessage(channel byte, data []byte) {
	if !s.supportsExt(cpeExtPluginMessages) {
		return
	}
	_ = s.writePacket(encodePluginMessage(channel, data))
}

func (s *session) SetLightingMode(mode byte, locked bool) {
	if !s.supportsExt(cpeExtLightingMode) {
		return
	}
	_ = s.writePacket(encodeLightingMode(mode, locked))
}
