package classic

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"

	"github.com/solar-mc/solar/internal/world"
)

func encodeLevelDataPackets(level world.Level, fastMap bool) ([][]byte, error) {
	var raw bytes.Buffer

	if fastMap {
		writer, err := flate.NewWriter(&raw, flate.BestSpeed)
		if err != nil {
			return nil, fmt.Errorf("create raw deflate writer: %w", err)
		}
		if err := writeLevelPayload(writer, level); err != nil {
			return nil, err
		}
	} else {
		writer := gzip.NewWriter(&raw)
		if err := writeLevelPayload(writer, level); err != nil {
			return nil, err
		}
	}

	payload := raw.Bytes()
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
	return packets, nil
}

func writeLevelPayload(writer interface {
	Write([]byte) (int, error)
	Close() error
}, level world.Level) error {
	header := make([]byte, 4)
	header[0] = byte(level.Volume() >> 24)
	header[1] = byte(level.Volume() >> 16)
	header[2] = byte(level.Volume() >> 8)
	header[3] = byte(level.Volume())
	if _, err := writer.Write(header); err != nil {
		return fmt.Errorf("write map header: %w", err)
	}
	if _, err := writer.Write(level.Blocks); err != nil {
		return fmt.Errorf("write map blocks: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close map stream: %w", err)
	}
	return nil
}
