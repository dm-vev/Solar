package loadtest

import (
	"testing"

	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/protocol/wire"
)

func TestEncodeLoadtestSetBlock(t *testing.T) {
	t.Parallel()

	packet := encodeLoadtestSetBlock(1, 2, 3, 7)
	if packet[0] != wire.OpcodeSetBlockClient {
		t.Fatalf("opcode = %d, want %d", packet[0], wire.OpcodeSetBlockClient)
	}
	if packet[8] != 7 {
		t.Fatalf("block id = %d, want 7", packet[8])
	}
}

func TestEncodeLoadtestTeleport(t *testing.T) {
	t.Parallel()

	packet := encodeLoadtestTeleport(entity.Position{X: 4, Y: 5, Z: 6}, 11, 22)
	if packet[0] != wire.OpcodeEntityTeleport {
		t.Fatalf("opcode = %d, want %d", packet[0], wire.OpcodeEntityTeleport)
	}
	if packet[1] != 255 {
		t.Fatalf("entity id = %d, want 255", packet[1])
	}
	if packet[8] != 11 || packet[9] != 22 {
		t.Fatalf("rotation = %d,%d, want 11,22", packet[8], packet[9])
	}
}
