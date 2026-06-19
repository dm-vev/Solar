package world

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LoadLevel reads a world snapshot from disk.
func LoadLevel(path string) (Level, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Level{}, fmt.Errorf("read world %s: %w", path, err)
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var legacy Level
		if err := json.Unmarshal(data, &legacy); err != nil {
			return Level{}, fmt.Errorf("decode legacy world %s: %w", path, err)
		}
		if err := validateLevelBounds(legacy); err != nil {
			return Level{}, fmt.Errorf("validate legacy world %s: %w", path, err)
		}
		return normalizeLevel(legacy), nil
	}

	level, err := decodeLevel(data)
	if err != nil {
		return Level{}, fmt.Errorf("decode world %s: %w", path, err)
	}
	if err := validateLevelBounds(level); err != nil {
		return Level{}, fmt.Errorf("validate world %s: %w", path, err)
	}
	return normalizeLevel(level), nil
}

// Save writes the world snapshot to disk.
func (l Level) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create world directory for %s: %w", path, err)
	}

	data, err := encodeLevel(normalizeLevel(l))
	if err != nil {
		return fmt.Errorf("encode world %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create world temp for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write world temp %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close world temp %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace world %s: %w", path, err)
	}
	return nil
}

func encodeLevel(level Level) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(fileMagic)
	if err := buf.WriteByte(fileVersion); err != nil {
		return nil, fmt.Errorf("write world version: %w", err)
	}

	if err := binary.Write(&buf, binary.LittleEndian, uint16(len(level.Name))); err != nil {
		return nil, fmt.Errorf("write name length: %w", err)
	}
	if _, err := buf.WriteString(level.Name); err != nil {
		return nil, fmt.Errorf("write name: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(level.Width)); err != nil {
		return nil, fmt.Errorf("write width: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(level.Height)); err != nil {
		return nil, fmt.Errorf("write height: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(level.Length)); err != nil {
		return nil, fmt.Errorf("write length: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, int32(level.Spawn.X)); err != nil {
		return nil, fmt.Errorf("write spawn x: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, int32(level.Spawn.Y)); err != nil {
		return nil, fmt.Errorf("write spawn y: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, int32(level.Spawn.Z)); err != nil {
		return nil, fmt.Errorf("write spawn z: %w", err)
	}
	if err := buf.WriteByte(level.Spawn.Yaw); err != nil {
		return nil, fmt.Errorf("write spawn yaw: %w", err)
	}
	if err := buf.WriteByte(level.Spawn.Pitch); err != nil {
		return nil, fmt.Errorf("write spawn pitch: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint32(len(level.Blocks))); err != nil {
		return nil, fmt.Errorf("write block length: %w", err)
	}
	if _, err := buf.Write(level.Blocks); err != nil {
		return nil, fmt.Errorf("write blocks: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeLevel(data []byte) (Level, error) {
	reader := bytes.NewReader(data)

	if err := readMagic(reader); err != nil {
		return Level{}, err
	}
	if err := readVersion(reader); err != nil {
		return Level{}, err
	}

	name, err := readLevelName(reader)
	if err != nil {
		return Level{}, err
	}

	dims, err := readDimensions(reader)
	if err != nil {
		return Level{}, err
	}

	spawn, err := readSpawn(reader)
	if err != nil {
		return Level{}, err
	}

	blocks, err := readBlocks(reader, dims)
	if err != nil {
		return Level{}, err
	}

	return Level{
		Name:   name,
		Width:  dims[0],
		Height: dims[1],
		Length: dims[2],
		Blocks: blocks,
		Spawn:  spawn,
	}, nil
}

func readMagic(reader *bytes.Reader) error {
	magic := make([]byte, len(fileMagic))
	if _, err := io.ReadFull(reader, magic); err != nil {
		return fmt.Errorf("read magic: %w", err)
	}
	if string(magic) != fileMagic {
		return fmt.Errorf("invalid magic %q", magic)
	}
	return nil
}

func readVersion(reader *bytes.Reader) error {
	version, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("read version: %w", err)
	}
	if version != fileVersion {
		return fmt.Errorf("unsupported version %d", version)
	}
	return nil
}

func readLevelName(reader *bytes.Reader) (string, error) {
	nameLen, err := readUint16(reader)
	if err != nil {
		return "", err
	}
	name := make([]byte, nameLen)
	if _, err := io.ReadFull(reader, name); err != nil {
		return "", fmt.Errorf("read name: %w", err)
	}
	return string(name), nil
}

func readDimensions(reader *bytes.Reader) ([3]int, error) {
	width, err := readUint16(reader)
	if err != nil {
		return [3]int{}, err
	}
	height, err := readUint16(reader)
	if err != nil {
		return [3]int{}, err
	}
	length, err := readUint16(reader)
	if err != nil {
		return [3]int{}, err
	}
	return [3]int{int(width), int(height), int(length)}, nil
}

func readSpawn(reader *bytes.Reader) (Spawn, error) {
	x, err := readInt32(reader)
	if err != nil {
		return Spawn{}, err
	}
	y, err := readInt32(reader)
	if err != nil {
		return Spawn{}, err
	}
	z, err := readInt32(reader)
	if err != nil {
		return Spawn{}, err
	}
	yaw, err := reader.ReadByte()
	if err != nil {
		return Spawn{}, fmt.Errorf("read yaw: %w", err)
	}
	pitch, err := reader.ReadByte()
	if err != nil {
		return Spawn{}, fmt.Errorf("read pitch: %w", err)
	}
	return Spawn{X: int(x), Y: int(y), Z: int(z), Yaw: yaw, Pitch: pitch}, nil
}

func readBlocks(reader *bytes.Reader, dims [3]int) ([]byte, error) {
	blockLen, err := readUint32(reader)
	if err != nil {
		return nil, err
	}
	width, height, length := dims[0], dims[1], dims[2]
	volume := int64(width) * int64(height) * int64(length)
	if volume < 1 || volume > maxBlocks {
		return nil, fmt.Errorf("world volume %d exceeds limit %d", volume, maxBlocks)
	}
	if int64(blockLen) != volume {
		return nil, fmt.Errorf("block length %d does not match volume %d", blockLen, volume)
	}
	if int64(blockLen) > int64(reader.Len()) {
		return nil, fmt.Errorf("block length %d exceeds remaining data %d", blockLen, reader.Len())
	}
	blocks := make([]byte, blockLen)
	if _, err := io.ReadFull(reader, blocks); err != nil {
		return nil, fmt.Errorf("read blocks: %w", err)
	}
	return blocks, nil
}

func readUint16(reader *bytes.Reader) (uint16, error) {
	var value uint16
	if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
		return 0, fmt.Errorf("read uint16: %w", err)
	}
	return value, nil
}

func readUint32(reader *bytes.Reader) (uint32, error) {
	var value uint32
	if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
		return 0, fmt.Errorf("read uint32: %w", err)
	}
	return value, nil
}

func readInt32(reader *bytes.Reader) (int32, error) {
	var value int32
	if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
		return 0, fmt.Errorf("read int32: %w", err)
	}
	return value, nil
}
