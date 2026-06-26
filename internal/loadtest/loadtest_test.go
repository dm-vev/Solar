package loadtest

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/protocol/wire"
)

func TestLoginClassicAndCPE(t *testing.T) {
	t.Run("classic", func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()
		done := make(chan struct{})
		go func() {
			defer close(done)
			readHandshake(t, server)
			writeMOTDAndLevel(t, server, false)
		}()
		if err := login(client, "bot", false); err != nil {
			t.Fatalf("login classic: %v", err)
		}
		<-done
	})

	t.Run("cpe", func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()
		done := make(chan struct{})
		go func() {
			defer close(done)
			readHandshake(t, server)
			if _, err := server.Write(encodeExtInfo("Solar", 0)); err != nil {
				t.Errorf("write ext info: %v", err)
				return
			}
			readClientCPE(t, server)
			writeMOTDAndLevel(t, server, true)
		}()
		if err := login(client, "bot", true); err != nil {
			t.Fatalf("login cpe: %v", err)
		}
		<-done
	})
}

func TestRunScenariosAndEncoders(t *testing.T) {
	if got := encodeHandshake(7, "bot", true); got[0] != wire.OpcodeHandshake || got[130] != 0x42 {
		t.Fatalf("encodeHandshake = %v", got[:2])
	}
	if got := encodeExtEntry("FastMap", 1); got[0] != wire.OpcodeExtEntry {
		t.Fatalf("encodeExtEntry opcode = %d", got[0])
	}
	if got := encodeLoadtestMessage("bot", 1, 2); got[0] != wire.OpcodeMessage {
		t.Fatalf("encodeLoadtestMessage opcode = %d", got[0])
	}
	if got := encodeLoadtestTeleport(1, 2, 3, 4, 5); got[0] != wire.OpcodeEntityTeleport {
		t.Fatalf("encodeLoadtestTeleport opcode = %d", got[0])
	}
	if got := encodeLoadtestSetBlock(1, 2, 3, 4); got[0] != wire.OpcodeSetBlockClient {
		t.Fatalf("encodeLoadtestSetBlock opcode = %d", got[0])
	}

	for _, scenario := range []string{scenarioIdle, scenarioChat, scenarioMove, scenarioBlocks, scenarioMixed} {
		server, client := net.Pipe()
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, _ = io.Copy(io.Discard, server)
		}()
		if err := runScenario(ctx, client, scenario, "bot", 1); err != nil {
			t.Fatalf("runScenario(%s): %v", scenario, err)
		}
		cancel()
		client.Close()
		server.Close()
		<-done
	}

	server, client := net.Pipe()
	server.Close()
	if err := sendClientPacket(client, []byte{1}); err == nil {
		t.Fatal("sendClientPacket succeeded on closed pipe")
	}
	client.Close()
	if err := runScenario(context.Background(), nil, "unknown", "bot", 0); err == nil {
		t.Fatal("runScenario accepted unknown scenario")
	}
}

func TestRunFailsWhenDialFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := Run(ctx, Config{
		Address:        "127.0.0.1:1",
		Clients:        1,
		Duration:       200 * time.Millisecond,
		UsernamePrefix: "bot",
		Scenario:       scenarioIdle,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err == nil {
		t.Fatal("Run succeeded against closed local port")
	}
	if _, err := dialWithContext(ctx, "127.0.0.1:1"); err == nil {
		t.Fatal("dialWithContext succeeded against closed local port")
	}
}

func readHandshake(t *testing.T, conn net.Conn) {
	t.Helper()
	buf := make([]byte, 131)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Errorf("read handshake: %v", err)
	}
}

func writeMOTDAndLevel(t *testing.T, conn net.Conn, cpe bool) {
	t.Helper()
	motd := make([]byte, 131)
	motd[0] = wire.OpcodeHandshake
	if _, err := conn.Write(motd); err != nil {
		t.Errorf("write motd: %v", err)
		return
	}
	if cpe {
		if _, err := conn.Write([]byte{wire.OpcodeLevelInitialize, 0, 0, 0, 0}); err != nil {
			t.Errorf("write fastmap begin: %v", err)
			return
		}
	} else if _, err := conn.Write([]byte{wire.OpcodeLevelInitialize}); err != nil {
		t.Errorf("write level begin: %v", err)
		return
	}
	if _, err := conn.Write([]byte{wire.OpcodeLevelFinalize, 0, 4, 0, 4, 0, 4}); err != nil {
		t.Errorf("write level finalise: %v", err)
		return
	}
	if _, err := conn.Write(make([]byte, 10)); err != nil {
		t.Errorf("write teleport: %v", err)
	}
}

func readClientCPE(t *testing.T, conn net.Conn) {
	t.Helper()
	if _, err := io.ReadFull(conn, make([]byte, 67)); err != nil {
		t.Errorf("read client ext info: %v", err)
		return
	}
	for i := 0; i < 3; i++ {
		if _, err := io.ReadFull(conn, make([]byte, 69)); err != nil {
			t.Errorf("read client ext entry: %v", err)
			return
		}
	}
}
