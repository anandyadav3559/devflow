package scheduler

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
)

// StartIPCServer listens on a Unix socket for incoming commands and passes them to the handler.
func StartIPCServer(ctx context.Context, sockPath string, handler func(command string) string) error {
	// Remove any existing socket file
	if err := os.Remove(sockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing IPC socket: %w", err)
	}

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("failed to start IPC server: %w", err)
	}
	if err := os.Chmod(sockPath, 0600); err != nil {
		listener.Close()
		_ = os.Remove(sockPath)
		return fmt.Errorf("failed to secure IPC socket permissions: %w", err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
		_ = os.Remove(sockPath)
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Server closed or broken
				return
			}
			go handleIPCConnection(conn, handler)
		}
	}()

	return nil
}

func handleIPCConnection(conn net.Conn, handler func(string) string) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		cmd := strings.TrimSpace(scanner.Text())
		if cmd == "" {
			return
		}
		resp := handler(cmd)
		if resp != "" {
			if !strings.HasSuffix(resp, "\n") {
				resp += "\n"
			}
			_, _ = conn.Write([]byte(resp))
		}
	}
}

// SendIPCCommand sends a command to a running workflow daemon and returns its response.
func SendIPCCommand(sockPath string, command string) (string, error) {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if !strings.HasSuffix(command, "\n") {
		command += "\n"
	}

	if _, err := conn.Write([]byte(command)); err != nil {
		return "", err
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	var response strings.Builder
	for scanner.Scan() {
		response.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		return response.String(), err
	}
	return response.String(), nil
}
