package loadtest

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/protocol/wire"
)

const (
	scenarioIdle   = "idle"
	scenarioChat   = "chat"
	scenarioMove   = "move"
	scenarioBlocks = "blocks"
	scenarioMixed  = "mixed"
)

func runScenario(ctx context.Context, conn net.Conn, scenario, username string, index int) error {
	switch strings.ToLower(strings.TrimSpace(scenario)) {
	case "", scenarioIdle:
		<-ctx.Done()
		return nil
	case scenarioChat:
		return runChatScenario(ctx, conn, username, index)
	case scenarioMove:
		return runMoveScenario(ctx, conn, index)
	case scenarioBlocks:
		return runBlockScenario(ctx, conn, index)
	case scenarioMixed:
		return runMixedScenario(ctx, conn, username, index)
	default:
		return fmt.Errorf("unknown loadtest scenario %q", scenario)
	}
}

func runChatScenario(ctx context.Context, conn net.Conn, username string, index int) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	step := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := sendClientPacket(conn, encodeLoadtestMessage(username, index, step)); err != nil {
				return err
			}
			step++
		}
	}
}

func runMoveScenario(ctx context.Context, conn net.Conn, index int) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	step := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			position := entity.Position{
				X: 16 + index*2 + step,
				Y: 10,
				Z: 16 + step,
			}
			if err := sendClientPacket(conn, encodeLoadtestTeleport(position, byte((index+step)%256), byte((index+step*2)%256))); err != nil {
				return err
			}
			step++
		}
	}
}

func runBlockScenario(ctx context.Context, conn net.Conn, index int) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	step := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			blockID := byte((index + step) % 2)
			if err := sendClientPacket(conn, encodeLoadtestSetBlock(0, 0, 0, blockID)); err != nil {
				return err
			}
			step++
		}
	}
}

func runMixedScenario(ctx context.Context, conn net.Conn, username string, index int) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	step := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			switch step % 3 {
			case 0:
				if err := sendClientPacket(conn, encodeLoadtestMessage(username, index, step)); err != nil {
					return err
				}
			case 1:
				position := entity.Position{
					X: 16 + index + step,
					Y: 10,
					Z: 16 + step,
				}
				if err := sendClientPacket(conn, encodeLoadtestTeleport(position, 255, 0)); err != nil {
					return err
				}
			default:
				blockID := byte((index + step) % 2)
				if err := sendClientPacket(conn, encodeLoadtestSetBlock(0, 0, 0, blockID)); err != nil {
					return err
				}
			}
			step++
		}
	}
}

func sendClientPacket(conn net.Conn, packet []byte) error {
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(packet); err != nil {
		return fmt.Errorf("write loadtest packet: %w", err)
	}
	return nil
}

func encodeLoadtestMessage(username string, index, step int) []byte {
	packet := make([]byte, 66)
	packet[0] = wire.OpcodeMessage
	packet[1] = 0
	wire.WriteFixedString(packet[2:66], fmt.Sprintf("loadtest %s %d %d", username, index, step))
	return packet
}

func encodeLoadtestTeleport(pos entity.Position, yaw, pitch byte) []byte {
	packet := make([]byte, 10)
	packet[0] = wire.OpcodeEntityTeleport
	packet[1] = 255
	binary.BigEndian.PutUint16(packet[2:4], uint16(pos.X*wire.CoordScale))
	binary.BigEndian.PutUint16(packet[4:6], uint16(pos.Y*wire.CoordScale+wire.EyeHeight))
	binary.BigEndian.PutUint16(packet[6:8], uint16(pos.Z*wire.CoordScale))
	packet[8] = yaw
	packet[9] = pitch
	return packet
}

func encodeLoadtestSetBlock(x, y, z int, blockID byte) []byte {
	packet := make([]byte, 9)
	packet[0] = wire.OpcodeSetBlockClient
	binary.BigEndian.PutUint16(packet[1:3], uint16(x))
	binary.BigEndian.PutUint16(packet[3:5], uint16(y))
	binary.BigEndian.PutUint16(packet[5:7], uint16(z))
	packet[7] = 1
	packet[8] = blockID
	return packet
}
