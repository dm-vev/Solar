package cpe

// CPE provides access to ClassiCube Protocol Extension packets.
// Plugins can use this to send CPE-specific packets to players.
// All methods are no-ops if the player's client doesn't support
// the relevant CPE extension.
type CPE interface {
	// SetEnvColor sets an environment color slot.
	// slot: 0=sky, 1=cloud, 2=fog, 3=ambient, 4=diffuse
	SetEnvColor(slot byte, r, g, b byte)

	// SetWeather changes the weather type.
	// weather: 0=sunny, 1=raining, 2=snowing
	SetWeather(weather byte)

	// SetHackControl controls player abilities.
	SetHackControl(flying, noclip, speed, respawnHeight, thirdPerson bool)

	// SetClickDistance sets the player's reach distance in blocks.
	SetClickDistance(distance float64)

	// SetTextHotkey registers a hotkey on the client.
	// key is the key code (e.g. 'F'), action is the text to send.
	SetTextHotkey(key byte, action string, flags byte)

	// HoldThis makes the player hold a specific block.
	HoldThis(block byte, canChange bool)

	// SetBlockPermission sets per-block placement/deletion permissions.
	SetBlockPermission(block byte, allowPlace, allowDelete bool)

	// ChangeModel changes how an entity looks to this player.
	ChangeModel(entityID byte, model string)

	// SetMapAppearance changes the server's map URL/texture.
	SetMapAppearance(url string, sideLevel byte)

	// SetInventoryOrder changes the hotbar order of a block.
	SetInventoryOrder(block byte, order byte)

	// SetHotbar selects a block in the player's hotbar.
	SetHotbar(slot byte, block byte)

	// SetSpawnpoint updates the player's spawnpoint.
	SetSpawnpoint(x, y, z int, yaw, pitch byte)

	// SendEffect spawns a visual effect.
	// effectID is a registered effect ID.
	SendEffect(effectID, x, y, z byte)

	// SendSelection creates a visible selection box.
	SendSelection(id byte, label string, minX, minY, minZ, maxX, maxY, maxZ int, r, g, b byte)

	// RemoveSelection removes a selection box.
	RemoveSelection(id byte)

	// SetMapProperty sets a map environment property.
	// property: 0=sidesLevel, 1=edgeLevel, 2=cloudsLevel, 3=maxFog, 4=cloudsSpeed, 5=weatherSpeed, 6=weatherFade, 7=expFog
	SetMapProperty(property byte, value int)

	// SetEntityProperty sets a property on an entity.
	// property: 0=scaleX, 1=scaleY, 2=scaleZ, 3=modelRotation, 4=modelYaw
	SetEntityProperty(entityID byte, property byte, value int)

	// SendPluginMessage sends a custom plugin message on a channel.
	SendPluginMessage(channel byte, data []byte)

	// SetLightingMode changes the lighting mode.
	// mode: 0=classic, 1=fancy
	SetLightingMode(mode byte, locked bool)
}
