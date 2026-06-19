package event

import "github.com/solar-mc/solar/plugin/player"

// Entity event data types and global event instances.

// EntitySpawnedData — entity is being spawned to someone. Modifiable.
type EntitySpawnedData struct {
	Player player.Player
	Name   *string
	Model  *string
}

// EntityDespawnedData — entity is being despawned from someone
type EntityDespawnedData struct {
	Player   player.Player
	EntityID byte
}

// SendingModelData — model is being sent to a player. Modifiable.
type SendingModelData struct {
	Player player.Player
	Model  *string
}

var (
	OnEntitySpawned   = NewEvent[EntitySpawnedData]()
	OnEntityDespawned = NewEvent[EntityDespawnedData]()
	OnSendingModel    = NewEvent[SendingModelData]()
)
