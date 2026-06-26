package server

import (
	"testing"

	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/plugin"
)

func TestEntityManagerSpawnDespawnTeleport(t *testing.T) {
	entities := entity.NewManager()
	codec := classic.NewCodec("test", "", nil, nil, entities, nil)
	em := newEntityManager(codec, entities)

	id := em.Spawn(plugin.EntityInfo{Name: "NPC", X: 32, Y: 64, Z: 96, Model: "humanoid"})
	if id == 0 {
		t.Fatal("Spawn returned 0")
	}
	if got := em.Count(); got < 1 {
		t.Fatalf("Count = %d, want >= 1", got)
	}

	info, ok := em.Get(id)
	if !ok {
		t.Fatalf("Get(%d) not found", id)
	}
	if info.Name != "NPC" || info.Model != "humanoid" || info.X != 32 {
		t.Fatalf("Get returned %+v", info)
	}

	if !em.Teleport(id, 100, 200, 300, 10, 20) {
		t.Fatal("Teleport returned false")
	}
	info, _ = em.Get(id)
	if info.X != 100 || info.Y != 200 || info.Z != 300 || info.Yaw != 10 || info.Pitch != 20 {
		t.Fatalf("after Teleport, Get returned %+v", info)
	}

	if !em.Despawn(id) {
		t.Fatal("Despawn returned false")
	}
	if _, ok := em.Get(id); ok {
		t.Fatal("Get found entity after Despawn")
	}
	if em.Despawn(id) {
		t.Fatal("Despawn on missing id returned true")
	}
	if em.Teleport(id, 0, 0, 0, 0, 0) {
		t.Fatal("Teleport on missing id returned true")
	}
}

func TestEntityManagerSpawnUsesSharedEntityIDSpace(t *testing.T) {
	entities := entity.NewManager()
	playerID, ok := entities.Add("player", entity.Position{})
	if !ok {
		t.Fatal("player Add returned false")
	}

	codec := classic.NewCodec("test", "", nil, nil, entities, nil)
	em := newEntityManager(codec, entities)
	npcID := em.Spawn(plugin.EntityInfo{Name: "NPC", X: 32, Y: 64, Z: 96, Model: "humanoid"})
	if npcID == 0 {
		t.Fatal("Spawn returned 0")
	}
	if npcID == byte(playerID) {
		t.Fatalf("plugin entity ID %d collided with player ID %d", npcID, playerID)
	}
	if _, ok := entities.Get(uint32(npcID)); !ok {
		t.Fatalf("shared entity manager does not contain plugin ID %d", npcID)
	}
}
