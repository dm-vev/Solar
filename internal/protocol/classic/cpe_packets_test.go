package classic

import "testing"

func TestCPEPacketSizes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		pkt  []byte
		want int
	}{
		{"ClickDistance", encodeClickDistance(5), 3},
		{"CustomBlockSupport", encodeCustomBlockSupportLevel(1), 2},
		{"HoldThis", encodeHoldThis(7, false), 3},
		{"TextHotkey", encodeTextHotkey("label", "input", 65, 0), 134},
		{"ExtAddEntity", encodeExtAddEntity(1, "skin", "name"), 130},
		{"EnvColor", encodeEnvColor(0, 100, 200, 50), 8},
		{"MakeSelection", encodeMakeSelection(0, "sel", 0, 0, 0, 10, 10, 10, 255, 0, 0, 128), 86},
		{"RemoveSelection", encodeRemoveSelection(0), 2},
		{"SetBlockPermission", encodeSetBlockPermission(7, true, true), 4},
		{"ChangeModel", encodeChangeModel(1, "human"), 66},
		{"MapAppearance", encodeMapAppearance("", 7, 8, 64), 69},
		{"MapAppearanceV2", encodeMapAppearanceV2("", 7, 8, 64, 128, 256), 73},
		{"EnvWeatherType", encodeEnvWeatherType(1), 2},
		{"HackControl", encodeHackControl(true, true, true, true, true, 256), 8},
		{"ExtAddEntity2", encodeExtAddEntity2(1, "name", "skin", 10, 20, 30, 0, 0), 138},
		{"SetTextColor", encodeSetTextColor(255, 0, 0, 255, 0), 6},
		{"SetMapEnvURL", encodeSetMapEnvURL("https://example.com"), 65},
		{"SetMapEnvURLV2", encodeSetMapEnvURLV2("https://example.com/long-url"), 129},
		{"SetMapEnvProperty", encodeSetMapEnvProperty(0, 100), 6},
		{"SetEntityProperty", encodeSetEntityProperty(1, 0, 100), 7},
		{"SetInventoryOrder", encodeSetInventoryOrder(7, 8), 3},
		{"SetHotbar", encodeSetHotbar(7, 0), 3},
		{"SetSpawnpoint", encodeSetSpawnpoint(10, 20, 30, 0, 0), 9},
		{"VelocityControl", encodeVelocityControl(0, 1, 0, 0, 0, 0), 16},
		{"DefineEffect", encodeDefineEffect(0, 0, 0, 1, 1, 255, 255, 255, 1, 1, 1, 0, 0, 0, 0, 0, 0, false, false, false, false, false), 36},
		{"SpawnEffect", encodeSpawnEffect(0, 1, 2, 3, 0, 0, 0), 26},
		{"DefineModel", encodeDefineModel(0, "model", true, true, true, true, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1, 1, 1, 1), 116},
		{"UndefineModel", encodeUndefineModel(0), 2},
		{"PluginMessage", encodePluginMessage(0, make([]byte, 64)), 66},
		{"EntityTeleportExt", encodeEntityTeleportExt(1, true, 0, true, false, 10, 20, 30, 0, 0), 11},
		{"LightingMode", encodeLightingMode(0, false), 3},
		{"CinematicGui", encodeCinematicGui(false, false, false, 0, 0, 0, 0, 0), 10},
		{"ToggleBlockList", encodeToggleBlockList(true), 2},
		{"DefineBlock", encodeDefineBlock(50, "custom", 0, 128, 0, 0, 0, true, 0, 0, 0, 0, 0, 0, 0), 80},
		{"UndefineBlock", encodeUndefineBlock(50), 2},
		{"DefineBlockExt", encodeDefineBlockExt(50, "custom", 0, 128, 0, 0, 0, true, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0), 85},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.pkt) != tc.want {
				t.Fatalf("%s packet size = %d, want %d", tc.name, len(tc.pkt), tc.want)
			}
		})
	}
}

func TestBulkBlockUpdateSize(t *testing.T) {
	t.Parallel()

	entries := []bulkBlockEntry{
		{Index: 0, BlockID: 1},
		{Index: 1, BlockID: 2},
		{Index: 2, BlockID: 3},
	}
	pkt := encodeBulkBlockUpdate(entries)
	// 1 opcode + 1 count + 3*4 entries + 1 lightFlagLen + 1 lightFlags
	want := 1 + 1 + 3*4 + 1 + 1
	if len(pkt) != want {
		t.Fatalf("BulkBlockUpdate size = %d, want %d", len(pkt), want)
	}
}
