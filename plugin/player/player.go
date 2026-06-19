// Package player defines the Player interface that plugins use to
// interact with an online player. The concrete implementation is the
// server's internal session type; plugins only see this interface.
package player

import "github.com/solar-mc/solar/plugin/cpe"

// Player is the interface plugins use to interact with an online player.
// The concrete implementation is the server's internal session type;
// plugins only see this interface.
type Player interface {
	// Name returns the player's username.
	Name() string

	// Message sends a chat message to this player only.
	Message(msg string)

	// Teleport moves the player to the given block coordinates.
	// Returns false if the entity is not found.
	Teleport(x, y, z int, yaw, pitch byte) bool

	// Kick disconnects the player with a reason message.
	Kick(reason string)

	// IsOperator reports whether the player has admin privileges.
	IsOperator() bool

	// Position returns the player's current position in wire units
	// (1/32 block). Divide by 32 for block coordinates.
	Position() (x, y, z int)

	// SetBlock changes a block in the world and broadcasts the change
	// to all players. Returns false if the coordinates are out of bounds.
	SetBlock(x, y, z int, block byte) bool

	// SupportsCPE reports whether the client supports the given CPE
	// extension name.
	SupportsCPE(extName string) bool

	// SendCpeMessage sends a CPE typed message (status bars, bottom-right, etc).
	// messageType: 0=chat, 1=status1, 2=status2, 3=status3, 4=bottomRight1, 5=bottomRight2, 6=announcement
	SendCpeMessage(messageType byte, msg string)

	// SetColor changes the player's name color (Classic color code like "&e").
	SetColor(color string)

	// Color returns the player's current color code.
	Color() string

	// SetModel changes the player's entity model (e.g. "humanoid", "chicken", "creeper").
	SetModel(model string)

	// Model returns the player's current model.
	Model() string

	// SetSkin changes the player's skin URL.
	SetSkin(skinURL string)

	// IsHidden reports whether the player is hidden from other players.
	IsHidden() bool

	// SetHidden hides or reveals the player.
	SetHidden(hidden bool)

	// IsMuted reports whether the player is muted.
	IsMuted() bool

	// SetMuted mutes or unmutes the player.
	SetMuted(muted bool)

	// IsFrozen reports whether the player is frozen (can't move).
	IsFrozen() bool

	// SetFrozen freezes or unfreezes the player.
	SetFrozen(frozen bool)

	// IsAfk reports whether the player is marked AFK.
	IsAfk() bool

	// IP returns the player's remote IP address.
	IP() string

	// EntityID returns the player's server-side entity ID.
	EntityID() uint32

	// Yaw returns the player's current yaw rotation.
	Yaw() byte

	// Pitch returns the player's current pitch rotation.
	Pitch() byte

	// SendBlockChange sends a block update to this player only (no world change).
	SendBlockChange(x, y, z int, block byte)

	// SendRawPacket sends raw bytes to the client. For advanced CPE usage.
	SendRawPacket(data []byte)

	// Respawn respawns the player at the world spawn point.
	Respawn()

	// AllowBuild reports whether the player can place/break blocks.
	AllowBuild() bool

	// SetAllowBuild controls whether the player can place/break blocks.
	SetAllowBuild(allowed bool)

	// CPE returns the CPE packet sender for this player.
	// Returns nil if the player doesn't support any CPE extensions.
	CPE() cpe.CPE

	// ChangeBlock places or removes a block as if the player did it.
	// Unlike SetBlock, this fires the OnBlockChange event and respects
	// the player's build permissions.
	ChangeBlock(x, y, z int, block byte) bool

	// RevertBlock sends the current server-side block at the given
	// coordinates back to the client, reverting any client-side change.
	RevertBlock(x, y, z int)

	// MakeSelection creates a visible selection box on this player's client.
	// id is 0-255, label is shown to the player, min/max define the box.
	MakeSelection(id byte, label string, minX, minY, minZ, maxX, maxY, maxZ int, r, g, b byte) bool
	// ClearSelection removes a selection box.
	ClearSelection(id byte) bool
}
