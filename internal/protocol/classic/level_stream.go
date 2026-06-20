// level_stream.go encodes the level data stream for transmission.
//
// The Classic protocol sends level data as a sequence of chunks:
//   1. LevelInitialize (opcode 0x02) — optionally with volume for FastMap
//   2. LevelDataChunk (opcode 0x03) — repeated, each up to 1024 bytes
//   3. LevelFinalize (opcode 0x04) — level dimensions
//
// The block array is gzip-compressed (standard) or raw-deflate (FastMap
// CPE extension). The Manager.LevelStream method handles compression
// and caching; this file only handles the chunk splitting.

package classic

import (
	"github.com/solar-mc/solar/internal/world"
)

func encodeLevelDataPackets(stream world.LevelStream) [][]byte {
	payload := stream.Payload
	packets := make([][]byte, 0, (len(payload)+maxChunkLen-1)/maxChunkLen)
	for len(payload) > 0 {
		chunkLen := len(payload)
		if chunkLen > maxChunkLen {
			chunkLen = maxChunkLen
		}

		packet := make([]byte, 1028)
		packet[0] = opcodeLevelData
		packet[1] = byte(chunkLen >> 8)
		packet[2] = byte(chunkLen)
		copy(packet[3:], payload[:chunkLen])
		packets = append(packets, packet)
		payload = payload[chunkLen:]
	}
	return packets
}
