package world

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/generator"
)

func TestLevelSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	want := Level{
		Name:   "arena",
		Width:  2,
		Height: 3,
		Length: 4,
		Blocks: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
		Spawn: Spawn{
			X:     64,
			Y:     96,
			Z:     128,
			Yaw:   17,
			Pitch: 29,
		},
	}

	path := filepath.Join(t.TempDir(), "main.json")
	if err := want.Save(path); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !bytes.HasPrefix(raw, []byte(fileMagic)) {
		t.Fatalf("saved world prefix = %q, want %q", raw[:4], fileMagic)
	}

	got, err := LoadLevel(path)
	if err != nil {
		t.Fatalf("LoadLevel returned error: %v", err)
	}

	if got.Name != want.Name || got.Width != want.Width || got.Height != want.Height || got.Length != want.Length {
		t.Fatalf("loaded level mismatch: got %+v want %+v", got, want)
	}
	if got.Spawn != want.Spawn {
		t.Fatalf("spawn mismatch: got %+v want %+v", got.Spawn, want.Spawn)
	}
	if !bytes.Equal(got.Blocks, want.Blocks) {
		t.Fatalf("blocks mismatch: got %v want %v", got.Blocks, want.Blocks)
	}
}

func TestLoadLevelRejectsOversizedBlockPayload(t *testing.T) {
	t.Parallel()

	var data bytes.Buffer
	data.WriteString(fileMagic)
	data.WriteByte(fileVersion)
	_ = binary.Write(&data, binary.LittleEndian, uint16(4))
	data.WriteString("main")
	_ = binary.Write(&data, binary.LittleEndian, uint16(1))
	_ = binary.Write(&data, binary.LittleEndian, uint16(1))
	_ = binary.Write(&data, binary.LittleEndian, uint16(1))
	_ = binary.Write(&data, binary.LittleEndian, int32(0))
	_ = binary.Write(&data, binary.LittleEndian, int32(0))
	_ = binary.Write(&data, binary.LittleEndian, int32(0))
	data.WriteByte(0)
	data.WriteByte(0)
	_ = binary.Write(&data, binary.LittleEndian, uint32(MaxBlocks()+1))

	path := filepath.Join(t.TempDir(), "oversized.world")
	if err := os.WriteFile(path, data.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if _, err := LoadLevel(path); err == nil {
		t.Fatal("LoadLevel returned nil error for oversized payload")
	}
}

func TestManagerLoadSaveRoundTrip(t *testing.T) {
	t.Parallel()

	want := Level{
		Name:   "play",
		Width:  1,
		Height: 1,
		Length: 2,
		Blocks: []byte{7, 8},
		Spawn: Spawn{
			X:     16,
			Y:     10,
			Z:     16,
			Yaw:   1,
			Pitch: 2,
		},
	}

	path := filepath.Join(t.TempDir(), "world.json")
	mgr := NewManager()
	mgr.SetCurrent(want)
	if err := mgr.Save(path); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	reloaded := NewManager()
	if err := reloaded.Load(path); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	got := reloaded.Current()
	if got.Name != want.Name || got.Width != want.Width || got.Height != want.Height || got.Length != want.Length {
		t.Fatalf("loaded level mismatch: got %+v want %+v", got, want)
	}
	if got.Spawn != want.Spawn {
		t.Fatalf("spawn mismatch: got %+v want %+v", got.Spawn, want.Spawn)
	}
	if !bytes.Equal(got.Blocks, want.Blocks) {
		t.Fatalf("blocks mismatch: got %v want %v", got.Blocks, want.Blocks)
	}
}

func TestManagerSpecialBlocksSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	mgr.SetCurrent(Level{Name: "special", Width: 4, Height: 4, Length: 4, Blocks: make([]byte, 64)})
	if !mgr.SetBlock(1, 2, 3, blocks.MBWhite) || !mgr.SetSpecialBlock(1, 2, 3, &blocks.SpecialEntry{
		Type:    blocks.SpecialMessage,
		Message: "hello",
	}) {
		t.Fatal("set message block")
	}

	path := filepath.Join(t.TempDir(), "special.swld")
	if err := mgr.Save(path); err != nil {
		t.Fatal(err)
	}
	reloaded := NewManager()
	if err := reloaded.Load(path); err != nil {
		t.Fatal(err)
	}
	entry := reloaded.SpecialBlockAt(1, 2, 3)
	if entry == nil || entry.Type != blocks.SpecialMessage || entry.Message != "hello" {
		t.Fatalf("special block = %+v", entry)
	}
	if !reloaded.SetBlock(1, 2, 3, blocks.Stone) || reloaded.SpecialBlockAt(1, 2, 3) != nil {
		t.Fatal("replacing physical block did not remove metadata")
	}
}

func TestManagerTickCount(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	mgr.Tick()
	mgr.Tick()

	if got := mgr.TickCount(); got != 2 {
		t.Fatalf("TickCount = %d, want 2", got)
	}
}

func TestManagerSetSpawn(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	mgr.SetSpawn(Spawn{X: 11, Y: 22, Z: 33, Yaw: 44, Pitch: 55})

	got := mgr.Current()
	if got.Spawn.X != 11 || got.Spawn.Y != 22 || got.Spawn.Z != 33 {
		t.Fatalf("spawn position = %+v, want 11,22,33", got.Spawn)
	}
	if got.Spawn.Yaw != 44 || got.Spawn.Pitch != 55 {
		t.Fatalf("spawn rotation = %+v, want 44,55", got.Spawn)
	}
}

func TestFromGeneratorLevelAndMaxBlocks(t *testing.T) {
	t.Parallel()

	oldMax := MaxBlocks()
	t.Cleanup(func() { SetMaxBlocks(int(oldMax)) })

	SetMaxBlocks(12345)
	if got := MaxBlocks(); got != 12345 {
		t.Fatalf("MaxBlocks = %d, want 12345", got)
	}
	SetMaxBlocks(0)
	if got := MaxBlocks(); got != 12345 {
		t.Fatalf("MaxBlocks after zero set = %d, want unchanged 12345", got)
	}

	src := &generator.Level{
		Name:   "gen",
		Width:  2,
		Height: 2,
		Length: 2,
		Blocks: []byte{1, 2, 3},
		Spawn:  generator.Spawn{X: 1, Y: 2, Z: 3, Yaw: 4, Pitch: 5},
	}
	got := FromGeneratorLevel(src)
	if got.Name != src.Name || got.Width != src.Width || got.Height != src.Height || got.Length != src.Length {
		t.Fatalf("converted dimensions = %+v", got)
	}
	if got.Spawn != (Spawn{X: 1, Y: 2, Z: 3, Yaw: 4, Pitch: 5}) {
		t.Fatalf("converted spawn = %+v", got.Spawn)
	}
	if !bytes.Equal(got.Blocks, src.Blocks) {
		t.Fatalf("converted blocks = %v, want %v", got.Blocks, src.Blocks)
	}
}

func TestManagerAsyncSaveLoad(t *testing.T) {
	t.Parallel()

	want := Level{
		Name:   "async",
		Width:  1,
		Height: 1,
		Length: 1,
		Blocks: []byte{9},
		Spawn: Spawn{
			X: 1, Y: 2, Z: 3,
		},
	}
	path := filepath.Join(t.TempDir(), "async.world")

	mgr := NewManager()
	mgr.SetCurrent(want)
	if err := <-mgr.SaveAsync(path); err != nil {
		t.Fatalf("SaveAsync returned error: %v", err)
	}

	reloaded := NewManager()
	if err := <-reloaded.LoadAsync(path); err != nil {
		t.Fatalf("LoadAsync returned error: %v", err)
	}
	got := reloaded.Current()
	if got.Name != want.Name || got.Width != want.Width || got.Height != want.Height || got.Length != want.Length {
		t.Fatalf("reloaded level mismatch: got %+v want %+v", got, want)
	}
	if !bytes.Equal(got.Blocks, want.Blocks) {
		t.Fatalf("reloaded blocks mismatch: got %v want %v", got.Blocks, want.Blocks)
	}
}

func TestBootstrapLevelIsFlat(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	level := mgr.Current()

	if level.Width != bootstrapWidth || level.Height != bootstrapHeight || level.Length != bootstrapLength {
		t.Fatalf("bootstrap level dims = %dx%dx%d, want %dx%dx%d",
			level.Width, level.Height, level.Length, bootstrapWidth, bootstrapHeight, bootstrapLength)
	}

	if level.Spawn.X != bootstrapWidth/2 || level.Spawn.Y != (bootstrapHeight*3)/4 || level.Spawn.Z != bootstrapLength/2 {
		t.Fatalf("bootstrap spawn = %+v, want center at 3/4 height", level.Spawn)
	}

	if got := level.Blocks[0]; got != classicDirt {
		t.Fatalf("ground block = %d, want dirt", got)
	}

	surfaceIndex := ((bootstrapHeight/2 - 1) * bootstrapWidth * bootstrapLength)
	if got := level.Blocks[surfaceIndex]; got != classicGrass {
		t.Fatalf("surface block = %d, want grass", got)
	}
}

func TestManagerConcurrentBlockAccess(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	mgr.SetCurrent(Level{
		Name:   "arena",
		Width:  4,
		Height: 4,
		Length: 4,
		Blocks: make([]byte, 64),
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			mgr.SetBlock(i%4, 0, 0, byte(i))
			mgr.Tick()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_, _ = mgr.BlockAt(i%4, 0, 0)
			_ = mgr.Current()
		}
	}()

	wg.Wait()

	if got := mgr.TickCount(); got != 1000 {
		t.Fatalf("TickCount = %d, want 1000", got)
	}
}

func TestManagerSpawn(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	want := Spawn{X: 11, Y: 22, Z: 33, Yaw: 44, Pitch: 55}
	mgr.SetSpawn(want)

	got := mgr.Spawn()
	if got != want {
		t.Fatalf("Spawn() = %+v, want %+v", got, want)
	}
}

func TestManagerConcurrentSpawnAndSetCurrent(t *testing.T) {
	t.Parallel()

	mgr := NewManager()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			mgr.SetSpawn(Spawn{X: i, Y: i, Z: i})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = mgr.Spawn()
		}
	}()

	wg.Wait()
}
