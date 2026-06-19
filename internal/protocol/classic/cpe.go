package classic

import (
	"encoding/binary"
	"fmt"
	"io"
)

// CPE extension names — matches the ClassiCube CPE specification.
const (
	cpeExtClickDistance       = "ClickDistance"
	cpeExtCustomBlocks        = "CustomBlocks"
	cpeExtHeldBlock           = "HeldBlock"
	cpeExtTextHotkey          = "TextHotKey"
	cpeExtPlayerListName      = "ExtPlayerList"
	cpeExtEnvColors           = "EnvColors"
	cpeExtSelectionCuboid     = "SelectionCuboid"
	cpeExtBlockPermissions    = "BlockPermissions"
	cpeExtChangeModel         = "ChangeModel"
	cpeExtEnvMapAppearance    = "EnvMapAppearance"
	cpeExtEnvWeatherType      = "EnvWeatherType"
	cpeExtHackControl         = "HackControl"
	cpeExtEmoteFix            = "EmoteFix"
	cpeExtMessageTypes        = "MessageTypes"
	cpeExtLongerMessages      = "LongerMessages"
	cpeExtFullCP437           = "FullCP437"
	cpeExtBlockDefinitions    = "BlockDefinitions"
	cpeExtBlockDefinitionsExt = "BlockDefinitionsExt"
	cpeExtTextColors          = "TextColors"
	cpeExtBulkBlockUpdate     = "BulkBlockUpdate"
	cpeExtEnvMapAspect        = "EnvMapAspect"
	cpeExtPlayerClick         = "PlayerClick"
	cpeExtEntityProperty      = "EntityProperty"
	cpeExtExtEntityPositions  = "ExtEntityPositions"
	cpeExtTwoWayPingName      = "TwoWayPing"
	cpeExtInventoryOrder      = "InventoryOrder"
	cpeExtInstantMOTD         = "InstantMOTD"
	cpeExtFastMapName         = "FastMap"
	cpeExtExtTextures         = "ExtendedTextures"
	cpeExtSetHotbar           = "SetHotbar"
	cpeExtSetSpawnpoint       = "SetSpawnpoint"
	cpeExtVelocityControl     = "VelocityControl"
	cpeExtCustomParticles     = "CustomParticles"
	cpeExtCustomModels        = "CustomModels"
	cpeExtPluginMessages      = "PluginMessages"
	cpeExtExtEntityTeleport   = "ExtEntityTeleport"
	cpeExtLightingMode        = "LightingMode"
	cpeExtCinematicGui        = "CinematicGui"
	cpeExtNotifyAction        = "NotifyAction"
	cpeExtToggleBlockList     = "ToggleBlockList"
)

// serverExtensions lists every CPE extension the server advertises.
// Version is the highest server-side version supported.
var serverExtensions = []struct {
	name    string
	version uint32
}{
	{cpeExtClickDistance, 1},
	{cpeExtCustomBlocks, 1},
	{cpeExtHeldBlock, 1},
	{cpeExtTextHotkey, 1},
	{cpeExtPlayerListName, 2},
	{cpeExtEnvColors, 1},
	{cpeExtSelectionCuboid, 1},
	{cpeExtBlockPermissions, 1},
	{cpeExtChangeModel, 1},
	{cpeExtEnvMapAppearance, 2},
	{cpeExtEnvWeatherType, 1},
	{cpeExtHackControl, 1},
	{cpeExtEmoteFix, 1},
	{cpeExtMessageTypes, 1},
	{cpeExtLongerMessages, 1},
	{cpeExtFullCP437, 1},
	{cpeExtBlockDefinitions, 1},
	{cpeExtBlockDefinitionsExt, 2},
	{cpeExtTextColors, 1},
	{cpeExtBulkBlockUpdate, 1},
	{cpeExtEnvMapAspect, 2},
	{cpeExtPlayerClick, 1},
	{cpeExtEntityProperty, 1},
	{cpeExtExtEntityPositions, 1},
	{cpeExtTwoWayPingName, 1},
	{cpeExtInventoryOrder, 1},
	{cpeExtInstantMOTD, 1},
	{cpeExtFastMapName, 1},
	{cpeExtExtTextures, 1},
	{cpeExtSetHotbar, 1},
	{cpeExtSetSpawnpoint, 1},
	{cpeExtVelocityControl, 1},
	{cpeExtCustomParticles, 1},
	{cpeExtCustomModels, 2},
	{cpeExtPluginMessages, 1},
	{cpeExtExtEntityTeleport, 1},
	{cpeExtLightingMode, 1},
	{cpeExtCinematicGui, 1},
	{cpeExtNotifyAction, 1},
	{cpeExtToggleBlockList, 1},
}

func (s *session) negotiateCPE() error {
	if err := s.writePacket(encodeExtInfo(s.serverName, len(serverExtensions))); err != nil {
		return fmt.Errorf("write ext info: %w", err)
	}
	for _, ext := range serverExtensions {
		if err := s.writePacket(encodeExtEntry(ext.name, ext.version)); err != nil {
			return fmt.Errorf("write ext entry %s: %w", ext.name, err)
		}
	}

	_, extCount, err := s.readExtInfo()
	if err != nil {
		return err
	}

	cpeExts := make(map[string]uint32, extCount)
	for i := 0; i < extCount; i++ {
		name, version, err := s.readExtEntry()
		if err != nil {
			return err
		}
		cpeExts[name] = version
	}
	s.setCPESupport(cpeExts)

	return nil
}

func (s *session) handleTwoWayPing() error {
	payload := make([]byte, 3)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read two way ping payload: %w", err)
	}
	if !s.supportsExt(cpeExtTwoWayPingName) {
		return nil
	}
	if payload[0] != 0 {
		return nil
	}
	return s.writePacket(encodeTwoWayPing(false, binary.BigEndian.Uint16(payload[1:3])))
}

func (s *session) readExtInfo() (string, int, error) {
	packet := make([]byte, 67)
	if _, err := io.ReadFull(s.reader, packet); err != nil {
		return "", 0, fmt.Errorf("read ext info payload: %w", err)
	}
	if packet[0] != opcodeExtInfo {
		return "", 0, fmt.Errorf("read ext info payload: unexpected opcode %d", packet[0])
	}

	return readFixedString(packet[1:65]), int(binary.BigEndian.Uint16(packet[65:67])), nil
}

func (s *session) readExtEntry() (string, uint32, error) {
	packet := make([]byte, 69)
	if _, err := io.ReadFull(s.reader, packet); err != nil {
		return "", 0, fmt.Errorf("read ext entry payload: %w", err)
	}
	if packet[0] != opcodeExtEntry {
		return "", 0, fmt.Errorf("read ext entry payload: unexpected opcode %d", packet[0])
	}

	return readFixedString(packet[1:65]), binary.BigEndian.Uint32(packet[65:69]), nil
}
