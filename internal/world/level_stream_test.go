package world

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"testing"
)

func TestLevelStreamReturnsDecompressiblePayload(t *testing.T) {
	t.Parallel()

	m := NewManager()
	stream, err := m.LevelStream(false)
	if err != nil {
		t.Fatalf("LevelStream: %v", err)
	}
	if len(stream.Payload) == 0 {
		t.Fatal("gzip payload is empty")
	}

	gr, err := gzip.NewReader(bytes.NewReader(stream.Payload))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()

	out, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if got, want := len(out), stream.Level.Volume()+4; got != want {
		t.Fatalf("decompressed size = %d, want %d", got, want)
	}

	volume := int(out[0])<<24 | int(out[1])<<16 | int(out[2])<<8 | int(out[3])
	if volume != stream.Level.Volume() {
		t.Fatalf("decoded volume = %d, want %d", volume, stream.Level.Volume())
	}
}

func TestLevelStreamFastMapReturnsDecompressiblePayload(t *testing.T) {
	t.Parallel()

	m := NewManager()
	stream, err := m.LevelStream(true)
	if err != nil {
		t.Fatalf("LevelStream: %v", err)
	}
	if len(stream.Payload) == 0 {
		t.Fatal("flate payload is empty")
	}

	fr := flate.NewReader(bytes.NewReader(stream.Payload))
	defer fr.Close()

	out, err := io.ReadAll(fr)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if got, want := len(out), stream.Level.Volume()+4; got != want {
		t.Fatalf("decompressed size = %d, want %d", got, want)
	}
}

func TestLevelStreamCachesPayloadAfterSetBlock(t *testing.T) {
	t.Parallel()

	m := NewManager()
	a, err := m.LevelStream(false)
	if err != nil {
		t.Fatalf("LevelStream: %v", err)
	}
	if len(a.Payload) == 0 {
		t.Fatal("first payload is empty")
	}

	m.SetBlock(0, 0, 0, 1)

	b, err := m.LevelStream(false)
	if err != nil {
		t.Fatalf("LevelStream after mutation: %v", err)
	}
	if bytes.Equal(a.Payload, b.Payload) {
		t.Fatal("payload was not regenerated after SetBlock")
	}
}

func TestLevelStreamCachesPayloadAcrossCalls(t *testing.T) {
	t.Parallel()

	m := NewManager()
	a, err := m.LevelStream(false)
	if err != nil {
		t.Fatalf("LevelStream: %v", err)
	}
	b, err := m.LevelStream(false)
	if err != nil {
		t.Fatalf("LevelStream: %v", err)
	}
	if !bytes.Equal(a.Payload, b.Payload) {
		t.Fatal("consecutive LevelStream calls should return cached payload")
	}
}
