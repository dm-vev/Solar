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
