package classic

import (
	"fmt"

	"github.com/solar-mc/solar/internal/blockdef"
)

// sendBlockDefinitions pushes all custom block definitions to the client
// before the world stream. Only called when the client supports the
// BlockDefinitions CPE extension.
func (s *session) sendBlockDefinitions() error {
	if s.blockDefs == nil || !s.supportsExt(cpeExtBlockDefinitions) {
		return nil
	}

	for _, def := range s.blockDefs.All() {
		if err := s.writePacket(s.encodeBlockDef(def)); err != nil {
			return fmt.Errorf("send block def %d: %w", def.ID, err)
		}
	}
	return nil
}

// encodeBlockDef chooses the appropriate packet variant based on which
// CPE extension version the client supports.
func (s *session) encodeBlockDef(def blockdef.BlockDefinition) []byte {
	if !def.IsSprite() && s.supportsExt(cpeExtBlockDefinitionsExt) {
		return encodeDefineBlockExt(
			def.ID, def.Name, def.CollideType, def.RawSpeed(),
			def.TopTex, def.RightTex, def.BottomTex,
			def.BlocksLight, def.WalkSound, def.BrightnessByte(),
			def.MinX, def.MinZ, def.MinY, def.MaxX, def.MaxZ, def.MaxY,
			def.BlockDraw, def.FogDensity, def.FogR, def.FogG, def.FogB,
		)
	}
	return encodeDefineBlock(
		def.ID, def.Name, def.CollideType, def.RawSpeed(),
		def.TopTex, def.RightTex, def.BottomTex,
		def.BlocksLight, def.WalkSound, def.BrightnessByte(),
		def.BlockDraw, def.FogDensity, def.FogR, def.FogG, def.FogB,
	)
}

// broadcastBlockDef sends a DefineBlock packet to all CPE-supporting peers.
func (s *session) broadcastBlockDef(def blockdef.BlockDefinition) {
	if s.room == nil {
		return
	}
	s.room.ForEachPeer(func(peer *session) {
		if !peer.supportsExt(cpeExtBlockDefinitions) {
			return
		}
		packet := peer.encodeBlockDef(def)
		if err := peer.writePacketNoCopy(packet); err != nil {
			s.logger.Debug("broadcast block def", "id", def.ID, "peer", peer.currentUsername(), "error", err)
		}
	})
}

// broadcastUndefineBlock sends an UndefineBlock packet to all CPE peers.
func (s *session) broadcastUndefineBlock(id byte) {
	if s.room == nil {
		return
	}
	packet := encodeUndefineBlock(id)
	s.room.ForEachPeer(func(peer *session) {
		if !peer.supportsExt(cpeExtBlockDefinitions) {
			return
		}
		if err := peer.writePacketNoCopy(packet); err != nil {
			s.logger.Debug("broadcast undefine block", "id", id, "peer", peer.currentUsername(), "error", err)
		}
	})
}
