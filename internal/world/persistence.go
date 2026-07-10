package world

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/solar-mc/solar/internal/blocks"
)

const maxSpecialData = 16 << 20

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
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync world temp %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close world temp %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace world %s: %w", path, err)
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("open world directory for sync %s: %w", path, err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync world directory for %s: %w", path, err)
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
	// Env section (version 2+). Preceded by env magic so old files
	// without it are handled gracefully.
	buf.WriteString("ENV1")
	binary.Write(&buf, binary.LittleEndian, level.Env.Weather)
	binary.Write(&buf, binary.LittleEndian, level.Env.EdgeLevel)
	binary.Write(&buf, binary.LittleEndian, level.Env.SidesLevel)
	binary.Write(&buf, binary.LittleEndian, level.Env.CloudsLevel)
	binary.Write(&buf, binary.LittleEndian, level.Env.MaxFog)
	if level.Env.ExpFog {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	binary.Write(&buf, binary.LittleEndian, level.Env.CloudsSpeed)
	binary.Write(&buf, binary.LittleEndian, level.Env.WeatherSpeed)
	binary.Write(&buf, binary.LittleEndian, level.Env.WeatherFade)
	binary.Write(&buf, binary.LittleEndian, level.Env.SkyboxHorSpeed)
	binary.Write(&buf, binary.LittleEndian, level.Env.SkyboxVerSpeed)
	for i := 0; i < 5; i++ {
		c := level.Env.Colors[i]
		buf.WriteByte(c.R)
		buf.WriteByte(c.G)
		buf.WriteByte(c.B)
		if c.Set {
			buf.WriteByte(1)
		} else {
			buf.WriteByte(0)
		}
	}
	buf.WriteByte(level.Env.LightingMode)
	if level.Env.LightingLock {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	// MOTD
	if len(level.Env.MOTD) > int(^uint16(0)) {
		return nil, fmt.Errorf("MOTD length %d exceeds uint16", len(level.Env.MOTD))
	}
	binary.Write(&buf, binary.LittleEndian, uint16(len(level.Env.MOTD)))
	buf.WriteString(level.Env.MOTD)
	special, err := json.Marshal(level.SpecialBlocks)
	if err != nil {
		return nil, fmt.Errorf("encode special blocks: %w", err)
	}
	if len(special) > maxSpecialData {
		return nil, fmt.Errorf("special block payload length %d exceeds limit %d", len(special), maxSpecialData)
	}
	buf.WriteString("SPCL")
	if err := binary.Write(&buf, binary.LittleEndian, uint32(len(special))); err != nil {
		return nil, fmt.Errorf("write special block length: %w", err)
	}
	if _, err := buf.Write(special); err != nil {
		return nil, fmt.Errorf("write special blocks: %w", err)
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

	env := readEnv(reader)
	special, err := readSpecialBlocks(reader, dims[0]*dims[1]*dims[2])
	if err != nil {
		return Level{}, err
	}
	return Level{
		Name:          name,
		Width:         dims[0],
		Height:        dims[1],
		Length:        dims[2],
		Blocks:        blocks,
		Spawn:         spawn,
		Env:           env,
		SpecialBlocks: special,
	}, nil
}

// readEnv reads the ENV1 section if present, otherwise returns defaults.
func readEnv(reader *bytes.Reader) Env {
	env := DefaultEnv()
	envMagic := make([]byte, 4)
	if n, _ := reader.Read(envMagic); n < 4 {
		return env // no env section, use defaults
	}
	if string(envMagic) != "ENV1" {
		_, _ = reader.Seek(-4, io.SeekCurrent)
		return env // unknown section, use defaults
	}
	var b byte
	binary.Read(reader, binary.LittleEndian, &env.Weather)
	binary.Read(reader, binary.LittleEndian, &env.EdgeLevel)
	binary.Read(reader, binary.LittleEndian, &env.SidesLevel)
	binary.Read(reader, binary.LittleEndian, &env.CloudsLevel)
	binary.Read(reader, binary.LittleEndian, &env.MaxFog)
	b, _ = reader.ReadByte()
	env.ExpFog = b == 1
	binary.Read(reader, binary.LittleEndian, &env.CloudsSpeed)
	binary.Read(reader, binary.LittleEndian, &env.WeatherSpeed)
	binary.Read(reader, binary.LittleEndian, &env.WeatherFade)
	binary.Read(reader, binary.LittleEndian, &env.SkyboxHorSpeed)
	binary.Read(reader, binary.LittleEndian, &env.SkyboxVerSpeed)
	for i := 0; i < 5; i++ {
		env.Colors[i].R, _ = reader.ReadByte()
		env.Colors[i].G, _ = reader.ReadByte()
		env.Colors[i].B, _ = reader.ReadByte()
		b, _ = reader.ReadByte()
		env.Colors[i].Set = b == 1
	}
	env.LightingMode, _ = reader.ReadByte()
	b, _ = reader.ReadByte()
	env.LightingLock = b == 1
	motdLen, err := readUint16(reader)
	if err == nil && motdLen > 0 {
		motd := make([]byte, motdLen)
		if _, err := io.ReadFull(reader, motd); err == nil {
			env.MOTD = string(motd)
		}
	}
	return env
}

func readSpecialBlocks(reader *bytes.Reader, volume int) ([]blocks.SpecialRecord, error) {
	if reader.Len() == 0 {
		return nil, nil
	}
	magic := make([]byte, 4)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return nil, fmt.Errorf("read special block magic: %w", err)
	}
	if string(magic) != "SPCL" {
		return nil, nil
	}
	size, err := readUint32(reader)
	if err != nil {
		return nil, err
	}
	if size > maxSpecialData || int64(size) > int64(reader.Len()) {
		return nil, fmt.Errorf("invalid special block payload length %d", size)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, fmt.Errorf("read special blocks: %w", err)
	}
	var records []blocks.SpecialRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("decode special blocks: %w", err)
	}
	if len(records) > volume {
		return nil, fmt.Errorf("special block count %d exceeds world volume %d", len(records), volume)
	}
	return records, nil
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
	limit := MaxBlocks()
	if volume < 1 || volume > limit {
		return nil, fmt.Errorf("world volume %d exceeds limit %d", volume, limit)
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
