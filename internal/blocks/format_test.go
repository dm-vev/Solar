package blocks

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/solar-mc/solar/plugin/blockdb"
)

// TestFileDB_ByteFormat verifies the binary file is byte-compatible
// with MCGalaxy's .cbdb format: magic, version, dims, entry layout.
func TestFileDB_ByteFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.cbdb")
	db, err := New(path, 128, 64, 128)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ts := time.Unix(epoch+100, 0) // 100 seconds after epoch
	db.Add(blockdb.Entry{
		PlayerID: 42,
		Time:     ts,
		X:        10, Y: 20, Z: 30,
		OldBlock: 0, NewBlock: 7,
		Flags: blockdb.ManualPlace,
	})
	if err := db.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Header: 16 bytes
	if len(data) < headerLen+entrySize {
		t.Fatalf("file size = %d, want >= %d", len(data), headerLen+entrySize)
	}

	// Magic
	if string(data[0:8]) != "CBDB_MCG" {
		t.Fatalf("magic = %q, want CBDB_MCG", string(data[0:8]))
	}

	// Version (LE u16 at offset 8)
	if v := binary.LittleEndian.Uint16(data[8:10]); v != 1 {
		t.Fatalf("version = %d, want 1", v)
	}

	// Dims (LE u16 at offsets 10, 12, 14)
	if w := binary.LittleEndian.Uint16(data[10:12]); w != 128 {
		t.Fatalf("width = %d, want 128", w)
	}
	if h := binary.LittleEndian.Uint16(data[12:14]); h != 64 {
		t.Fatalf("height = %d, want 64", h)
	}
	if l := binary.LittleEndian.Uint16(data[14:16]); l != 128 {
		t.Fatalf("length = %d, want 128", l)
	}

	// Entry at offset 16
	off := headerLen
	// PlayerID (LE i32)
	if pid := int32(binary.LittleEndian.Uint32(data[off : off+4])); pid != 42 {
		t.Fatalf("PlayerID = %d, want 42", pid)
	}
	// TimeDelta (LE i32)
	if td := int32(binary.LittleEndian.Uint32(data[off+4 : off+8])); td != 100 {
		t.Fatalf("TimeDelta = %d, want 100", td)
	}
	// PackedIndex (LE i32): x + width*(z + length*y) = 10 + 128*(30 + 128*20) = 10 + 128*(30+2560) = 10 + 128*2590 = 10 + 331520 = 331530
	expectedIdx := uint32(10 + 128*(30+128*20))
	if idx := binary.LittleEndian.Uint32(data[off+8 : off+12]); idx != expectedIdx {
		t.Fatalf("Index = %d, want %d", idx, expectedIdx)
	}
	// OldBlock, NewBlock
	if data[off+12] != 0 {
		t.Fatalf("OldBlock = %d, want 0", data[off+12])
	}
	if data[off+13] != 7 {
		t.Fatalf("NewBlock = %d, want 7", data[off+13])
	}
	// Flags (LE u16)
	if f := binary.LittleEndian.Uint16(data[off+14 : off+16]); f != uint16(blockdb.ManualPlace) {
		t.Fatalf("Flags = %d, want %d", f, uint16(blockdb.ManualPlace))
	}
}

func TestNameConverter(t *testing.T) {
	nc := NewNameConverter()

	id1 := nc.Get("Alice")
	id2 := nc.Get("Bob")
	id1Again := nc.Get("alice") // case-insensitive

	if id1 == 0 {
		t.Fatal("first ID should not be 0")
	}
	if id1 == id2 {
		t.Fatal("different players should have different IDs")
	}
	if id1 != id1Again {
		t.Fatalf("case-insensitive: alice=%d, Alice=%d", id1Again, id1)
	}
}

func TestNameConverterSet(t *testing.T) {
	nc := NewNameConverter()
	nc.Set("Alice", 42)
	if got := nc.Get("alice"); got != 42 {
		t.Fatalf("Get after Set = %d, want 42", got)
	}
	if got := nc.Get("Bob"); got != 43 {
		t.Fatalf("next assigned ID = %d, want 43", got)
	}
}
