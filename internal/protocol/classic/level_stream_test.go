package classic

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net"
	"testing"
)

type levelStreamResult struct {
	Begin    []byte
	Payload  []byte
	Finalise []byte
	Teleport []byte
}

func readLevelStream(t *testing.T, client net.Conn, fastMap bool) levelStreamResult {
	t.Helper()

	result := levelStreamResult{}
	beginSize := 1
	if fastMap {
		beginSize = 5
	}

	result.Begin = make([]byte, beginSize)
	if _, err := io.ReadFull(client, result.Begin); err != nil {
		t.Fatalf("read level begin: %v", err)
	}
	if result.Begin[0] != opcodeLevelInitialize {
		t.Fatalf("level begin opcode = %d, want %d", result.Begin[0], opcodeLevelInitialize)
	}

	var compressed bytes.Buffer
	for {
		var opcode [1]byte
		if _, err := io.ReadFull(client, opcode[:]); err != nil {
			t.Fatalf("read level stream opcode: %v", err)
		}

		switch opcode[0] {
		case opcodeLevelData:
			chunk := make([]byte, 1027)
			if _, err := io.ReadFull(client, chunk); err != nil {
				t.Fatalf("read level chunk: %v", err)
			}
			used := int(chunk[0])<<8 | int(chunk[1])
			if used > 1024 {
				t.Fatalf("level chunk used length = %d, want <= 1024", used)
			}
			compressed.Write(chunk[2 : 2+used])
		case opcodeLevelFinalize:
			tail := make([]byte, 6)
			if _, err := io.ReadFull(client, tail); err != nil {
				t.Fatalf("read level finalise: %v", err)
			}
			result.Finalise = append([]byte{opcodeLevelFinalize}, tail...)

			var teleportOpcode [1]byte
			if _, err := io.ReadFull(client, teleportOpcode[:]); err != nil {
				t.Fatalf("read teleport opcode: %v", err)
			}
			if teleportOpcode[0] != opcodeEntityTeleport {
				t.Fatalf("teleport opcode = %d, want %d", teleportOpcode[0], opcodeEntityTeleport)
			}

			teleportTail := make([]byte, 9)
			if _, err := io.ReadFull(client, teleportTail); err != nil {
				t.Fatalf("read teleport: %v", err)
			}
			result.Teleport = append([]byte{teleportOpcode[0]}, teleportTail...)

			payload, err := decompressLevelPayload(compressed.Bytes(), fastMap)
			if err != nil {
				t.Fatalf("decompress level payload: %v", err)
			}
			result.Payload = payload
			return result
		default:
			t.Fatalf("unexpected level opcode %d", opcode[0])
		}
	}
}

func decompressLevelPayload(compressed []byte, fastMap bool) ([]byte, error) {
	var reader io.ReadCloser
	if fastMap {
		reader = flate.NewReader(bytes.NewReader(compressed))
	} else {
		var err error
		reader, err = gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return nil, err
		}
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
