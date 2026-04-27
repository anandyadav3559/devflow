//go:build !windows

package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStartIPCServerSetsSecureSocketModeAndHandlesCommand(t *testing.T) {
	tmp := t.TempDir()
	sock := filepath.Join(tmp, "daemon.sock")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := StartIPCServer(ctx, sock, func(command string) string {
		if strings.HasPrefix(command, "START ") {
			return "OK"
		}
		return "UNKNOWN_COMMAND"
	}); err != nil {
		t.Fatalf("StartIPCServer() returned error: %v", err)
	}

	// Give the listener a brief moment to bind the socket.
	time.Sleep(50 * time.Millisecond)

	info, err := os.Stat(sock)
	if err != nil {
		t.Fatalf("expected socket file to exist: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Fatalf("expected socket mode 0600, got %#o", mode)
	}

	resp, err := SendIPCCommand(sock, "START api")
	if err != nil {
		t.Fatalf("SendIPCCommand() returned error: %v", err)
	}
	if strings.TrimSpace(resp) != "OK" {
		t.Fatalf("expected OK response, got %q", resp)
	}
}
