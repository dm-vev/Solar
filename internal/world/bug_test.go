// bug_test.go reproduces and verifies fixes for specific bugs.

package world

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// BUG: Env persistence round-trip — write level with custom Env, read back, verify.
func TestBug_EnvRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.swld")
	original := Level{
		Name:  "test",
		Width: 16, Height: 16, Length: 16,
		Blocks: make([]byte, 16*16*16),
		Spawn:  Spawn{X: 8, Y: 8, Z: 8, Yaw: 0, Pitch: 0},
		Env: Env{
			Weather:     1, // raining
			EdgeLevel:   10,
			CloudsLevel: 50,
			MaxFog:      20,
			ExpFog:      true,
			CloudsSpeed: 100,
			MOTD:        "Test Level",
			Colors: [5]EnvColor{
				{R: 100, G: 149, B: 237, Set: true}, // sky
				{R: 255, G: 255, B: 255, Set: true}, // cloud
				{Set: false},                        // fog (default)
				{Set: false},                        // ambient
				{Set: false},                        // diffuse
			},
			LightingMode: 1,
			LightingLock: true,
		},
	}

	if err := original.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadLevel(path)
	if err != nil {
		t.Fatalf("LoadLevel: %v", err)
	}

	if loaded.Env.Weather != 1 {
		t.Fatalf("Weather: got %d, want 1", loaded.Env.Weather)
	}
	if loaded.Env.EdgeLevel != 10 {
		t.Fatalf("EdgeLevel: got %d, want 10", loaded.Env.EdgeLevel)
	}
	if loaded.Env.CloudsLevel != 50 {
		t.Fatalf("CloudsLevel: got %d, want 50", loaded.Env.CloudsLevel)
	}
	if !loaded.Env.ExpFog {
		t.Fatal("ExpFog should be true")
	}
	if loaded.Env.CloudsSpeed != 100 {
		t.Fatalf("CloudsSpeed: got %d, want 100", loaded.Env.CloudsSpeed)
	}
	if loaded.Env.MOTD != "Test Level" {
		t.Fatalf("MOTD: got %q, want 'Test Level'", loaded.Env.MOTD)
	}
	if !loaded.Env.Colors[0].Set || loaded.Env.Colors[0].R != 100 {
		t.Fatalf("Sky color: Set=%v R=%d, want Set=true R=100", loaded.Env.Colors[0].Set, loaded.Env.Colors[0].R)
	}
	if loaded.Env.Colors[2].Set {
		t.Fatal("Fog color should not be set")
	}
	if loaded.Env.LightingMode != 1 {
		t.Fatalf("LightingMode: got %d, want 1", loaded.Env.LightingMode)
	}
	if !loaded.Env.LightingLock {
		t.Fatal("LightingLock should be true")
	}
}

// BUG: Level with empty MOTD should round-trip correctly.
func TestBug_EnvEmptyMOTD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.swld")
	lvl := Level{
		Name: "test", Width: 4, Height: 4, Length: 4,
		Blocks: make([]byte, 64),
		Env:    DefaultEnv(),
	}
	if err := lvl.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := LoadLevel(path)
	if err != nil {
		t.Fatalf("LoadLevel: %v", err)
	}
	if loaded.Env.MOTD != "" {
		t.Fatalf("MOTD: got %q, want empty", loaded.Env.MOTD)
	}
}

// BUG: Old level file without ENV1 section should get default Env.
func TestBug_OldLevelWithoutEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.swld")
	// Write a level, then truncate to remove ENV1 section.
	lvl := Level{
		Name: "test", Width: 4, Height: 4, Length: 4,
		Blocks: make([]byte, 64),
		Env:    DefaultEnv(),
	}
	if err := lvl.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Truncate file to just header + blocks (no ENV1).
	// Header=16, blocks=64, total=80 bytes.
	info, _ := os.Stat(path)
	if info.Size() > 80 {
		f, _ := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o644)
		// Re-encode just header + blocks.
		var buf bytes.Buffer
		buf.WriteString("SWLD")
		buf.WriteByte(1)
		binary.Write(&buf, binary.LittleEndian, uint16(len("test")))
		buf.WriteString("test")
		binary.Write(&buf, binary.LittleEndian, uint16(4))
		binary.Write(&buf, binary.LittleEndian, uint16(4))
		binary.Write(&buf, binary.LittleEndian, uint16(4))
		binary.Write(&buf, binary.LittleEndian, int32(0))
		binary.Write(&buf, binary.LittleEndian, int32(0))
		binary.Write(&buf, binary.LittleEndian, int32(0))
		buf.WriteByte(0) // yaw
		buf.WriteByte(0) // pitch
		binary.Write(&buf, binary.LittleEndian, uint32(64))
		buf.Write(make([]byte, 64))
		f.Write(buf.Bytes())
		f.Close()
	}
	loaded, err := LoadLevel(path)
	if err != nil {
		t.Fatalf("LoadLevel: %v", err)
	}
	// Should get default env.
	if loaded.Env.Weather != 0 {
		t.Fatalf("Weather: got %d, want 0 (default)", loaded.Env.Weather)
	}
}
