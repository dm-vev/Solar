package event

import "github.com/solar-mc/solar/plugin/player"

// Player event data types and global event instances.

// PlayerConnectData fires after a player has fully connected and spawned.
// Not cancelable (the player is already online).
type PlayerConnectData struct {
	Player player.Player
}

// PlayerDisconnectData fires before a player is removed from the server.
// Not cancelable.
type PlayerDisconnectData struct {
	Player player.Player
	Reason string
}

// PlayerStartConnectingData — after handshake, before CPE negotiation
type PlayerStartConnectingData struct{ Player player.Player }

// PlayerFinishConnectingData — after CPE handshake, before map send
type PlayerFinishConnectingData struct{ Player player.Player }

// PlayerChatData fires when a player sends a chat message.
// Message is mutable — handlers can modify the text.
// Cancelable — if cancelled, the message is not broadcast.
type PlayerChatData struct {
	Player  player.Player
	Message *string
}

// MessageReceivedData — player receives a message from server/other. Cancelable + modifiable.
type MessageReceivedData struct {
	Player  player.Player
	Message *string
}

// PlayerMoveData fires when a player moves.
// Cancelable — if cancelled, the movement is rejected.
type PlayerMoveData struct {
	Player player.Player
	X      int // new position in wire units (1/32 block)
	Y      int
	Z      int
	Yaw    byte
	Pitch  byte
}

// PlayerCommandData fires when a player runs a command.
// Cancelable — if cancelled, the command is not executed.
type PlayerCommandData struct {
	Player  player.Player
	Command string
	Args    string
}

// PlayerHelpData — player runs /help. Cancelable.
type PlayerHelpData struct {
	Player player.Player
	Target string
}

// PlayerActionData — player performs an action (AFK, Hide, Referee)
type PlayerActionData struct {
	Player  player.Player
	Action  string
	Message string
	Stealth bool
}

// PlayerDyingData — player is about to die. Cancelable.
type PlayerDyingData struct {
	Player player.Player
	Cause  byte
}

// PlayerDiedData — player has died. Cooldown is modifiable.
type PlayerDiedData struct {
	Player   player.Player
	Cause    byte
	Cooldown *int
}

// BlockChangeData fires when a player places or removes a block.
// Cancelable — if cancelled, the block change is rejected.
type BlockChangeData struct {
	Player  player.Player
	X       int
	Y       int
	Z       int
	Block   byte // 0 = air (removal)
	Placing bool
}

// BlockChangedData fires after a block has been changed.
// Not cancelable (the change already happened).
type BlockChangedData struct {
	Player  player.Player
	X       int
	Y       int
	Z       int
	Block   byte
	Placing bool
}

// PlayerClickData fires when a player clicks (CPE PlayerClick).
type PlayerClickData struct {
	Player   player.Player
	Button   byte // 0=left, 1=right, 2=middle
	Action   byte // 0=pressed, 1=released
	EntityID byte
	X        int
	Y        int
	Z        int
	Face     byte
}

// PlayerSpawnData fires when a player is spawning or respawning.
// Position/rotation are mutable.
type PlayerSpawnData struct {
	Player player.Player
	X      *int
	Y      *int
	Z      *int
	Yaw    *byte
	Pitch  *byte
}

// SentMapData — player has been sent a new map
type SentMapData struct{ Player player.Player }

// JoiningLevelData — player intends to join a map. Cancelable.
type JoiningLevelData struct {
	Player    player.Player
	LevelName string
}

// JoinedLevelData — player has spawned in a map
type JoinedLevelData struct {
	Player    player.Player
	LevelName string
	PrevLevel string
}

// SettingColorData — player color is being updated. Modifiable.
type SettingColorData struct {
	Player player.Player
	Color  *string
}

// GettingMotdData — MOTD is being retrieved for a player. Modifiable.
type GettingMotdData struct {
	Player player.Player
	Motd   *string
}

// SendingMotdData — MOTD is being sent to a player. Modifiable.
type SendingMotdData struct {
	Player player.Player
	Motd   *string
}

// GettingCanSeeData — checking if player can see another. Modifiable.
type GettingCanSeeData struct {
	Player player.Player
	Target player.Player
	CanSee *bool
}

// NotifyActionData — CPE NotifyAction (blocklist toggle, level saved, respawned, etc)
type NotifyActionData struct {
	Player player.Player
	Action byte
	Value  int
}

// NotifyPositionActionData — CPE NotifyPositionAction (respawn, setspawn)
type NotifyPositionActionData struct {
	Player player.Player
	Action byte
	X      int
	Y      int
	Z      int
}

var (
	OnPlayerConnect          = NewEvent[PlayerConnectData]()
	OnPlayerDisconnect       = NewEvent[PlayerDisconnectData]()
	OnPlayerStartConnecting  = NewEvent[PlayerStartConnectingData]()
	OnPlayerFinishConnecting = NewEvent[PlayerFinishConnectingData]()
	OnPlayerChat             = NewEvent[PlayerChatData]()
	OnMessageReceived        = NewEvent[MessageReceivedData]()
	OnPlayerMove             = NewEvent[PlayerMoveData]()
	OnPlayerCommand          = NewEvent[PlayerCommandData]()
	OnPlayerHelp             = NewEvent[PlayerHelpData]()
	OnPlayerAction           = NewEvent[PlayerActionData]()
	OnPlayerDying            = NewEvent[PlayerDyingData]()
	OnPlayerDied             = NewEvent[PlayerDiedData]()
	OnBlockChange            = NewEvent[BlockChangeData]()
	OnBlockChanged           = NewEvent[BlockChangedData]()
	OnPlayerClick            = NewEvent[PlayerClickData]()
	OnPlayerSpawn            = NewEvent[PlayerSpawnData]()
	OnSentMap                = NewEvent[SentMapData]()
	OnJoiningLevel           = NewEvent[JoiningLevelData]()
	OnJoinedLevel            = NewEvent[JoinedLevelData]()
	OnSettingColor           = NewEvent[SettingColorData]()
	OnGettingMotd            = NewEvent[GettingMotdData]()
	OnSendingMotd            = NewEvent[SendingMotdData]()
	OnGettingCanSee          = NewEvent[GettingCanSeeData]()
	OnNotifyAction           = NewEvent[NotifyActionData]()
	OnNotifyPositionAction   = NewEvent[NotifyPositionActionData]()
)
