package loadtest

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/solar-mc/solar/internal/protocol/wire"
)

func runClient(ctx context.Context, cfg Config, index int) error {
	username := fmt.Sprintf("%s-%03d", cfg.UsernamePrefix, index+1)

	conn, err := dialWithContext(ctx, cfg.Address)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	if err := login(conn, username, cfg.CPE); err != nil {
		_ = conn.Close()
		return err
	}

	drained := make(chan struct{})
	go func() {
		defer close(drained)
		_, _ = io.Copy(io.Discard, conn)
	}()

	err = runScenario(ctx, conn, cfg.Scenario, username, index)
	_ = conn.Close()
	<-drained
	if err != nil {
		return err
	}
	return nil
}

func login(conn net.Conn, username string, cpe bool) error {
	if _, err := conn.Write(encodeHandshake(7, username, cpe)); err != nil {
		return fmt.Errorf("write handshake: %w", err)
	}

	if cpe {
		if err := drainCPEHandshake(conn); err != nil {
			return err
		}
	}

	if err := readMOTD(conn); err != nil {
		return err
	}
	if err := drainLevel(conn, cpe); err != nil {
		return err
	}
	return nil
}

func encodeHandshake(version byte, username string, cpe bool) []byte {
	packet := make([]byte, 131)
	packet[0] = wire.OpcodeHandshake
	packet[1] = version
	wire.WriteFixedString(packet[2:66], username)
	wire.WriteFixedString(packet[66:130], "password")
	if cpe {
		packet[130] = 0x42
	}
	return packet
}

func drainCPEHandshake(conn net.Conn) error {
	serverExtInfo := make([]byte, 67)
	if _, err := io.ReadFull(conn, serverExtInfo); err != nil {
		return fmt.Errorf("read server ext info: %w", err)
	}
	if serverExtInfo[0] != wire.OpcodeExtInfo {
		return fmt.Errorf("server ext info opcode = %d, want %d", serverExtInfo[0], wire.OpcodeExtInfo)
	}
	count := int(binary.BigEndian.Uint16(serverExtInfo[65:67]))
	for i := 0; i < count; i++ {
		entry := make([]byte, 69)
		if _, err := io.ReadFull(conn, entry); err != nil {
			return fmt.Errorf("read server ext entry: %w", err)
		}
		if entry[0] != wire.OpcodeExtEntry {
			return fmt.Errorf("server ext entry opcode = %d, want %d", entry[0], wire.OpcodeExtEntry)
		}
	}

	extensions := []struct {
		name    string
		version uint32
	}{
		{name: "ExtPlayerList", version: 2},
		{name: "TwoWayPing", version: 1},
		{name: "FastMap", version: 1},
	}
	if _, err := conn.Write(encodeExtInfo("ClassiCube", len(extensions))); err != nil {
		return fmt.Errorf("write client ext info: %w", err)
	}
	for _, extension := range extensions {
		if _, err := conn.Write(encodeExtEntry(extension.name, extension.version)); err != nil {
			return fmt.Errorf("write client ext entry %s: %w", extension.name, err)
		}
	}
	return nil
}

func readMOTD(conn net.Conn) error {
	motd := make([]byte, 131)
	if _, err := io.ReadFull(conn, motd); err != nil {
		return fmt.Errorf("read motd: %w", err)
	}
	if motd[0] != wire.OpcodeHandshake {
		return fmt.Errorf("motd opcode = %d, want %d", motd[0], wire.OpcodeHandshake)
	}
	return nil
}

func drainLevel(conn net.Conn, cpe bool) error {
	if cpe {
		begin := make([]byte, 5)
		if _, err := io.ReadFull(conn, begin); err != nil {
			return fmt.Errorf("read fastmap begin: %w", err)
		}
		if begin[0] != wire.OpcodeLevelInitialize {
			return fmt.Errorf("level begin opcode = %d, want %d", begin[0], wire.OpcodeLevelInitialize)
		}
	} else {
		begin := make([]byte, 1)
		if _, err := io.ReadFull(conn, begin); err != nil {
			return fmt.Errorf("read level begin: %w", err)
		}
		if begin[0] != wire.OpcodeLevelInitialize {
			return fmt.Errorf("level begin opcode = %d, want %d", begin[0], wire.OpcodeLevelInitialize)
		}
	}

	for {
		opcode := make([]byte, 1)
		if _, err := io.ReadFull(conn, opcode); err != nil {
			return fmt.Errorf("read level stream opcode: %w", err)
		}
		switch opcode[0] {
		case wire.OpcodeLevelData:
			chunk := make([]byte, 1027)
			if _, err := io.ReadFull(conn, chunk); err != nil {
				return fmt.Errorf("read level chunk: %w", err)
			}
		case wire.OpcodeLevelFinalize:
			finalise := make([]byte, 6)
			if _, err := io.ReadFull(conn, finalise); err != nil {
				return fmt.Errorf("read level finalise: %w", err)
			}
			teleport := make([]byte, 10)
			if _, err := io.ReadFull(conn, teleport); err != nil {
				return fmt.Errorf("read teleport: %w", err)
			}
			return nil
		default:
			return fmt.Errorf("unexpected level opcode %d", opcode[0])
		}
	}
}

func encodeExtInfo(appName string, count int) []byte {
	packet := make([]byte, 67)
	packet[0] = wire.OpcodeExtInfo
	wire.WriteFixedString(packet[1:65], appName)
	packet[65] = byte(count >> 8)
	packet[66] = byte(count)
	return packet
}

func encodeExtEntry(name string, version uint32) []byte {
	packet := make([]byte, 69)
	packet[0] = wire.OpcodeExtEntry
	wire.WriteFixedString(packet[1:65], name)
	packet[65] = byte(version >> 24)
	packet[66] = byte(version >> 16)
	packet[67] = byte(version >> 8)
	packet[68] = byte(version)
	return packet
}
