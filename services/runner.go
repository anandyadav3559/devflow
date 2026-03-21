package services

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

// RunService starts a service — either detached (background process) or
// in a new terminal window, depending on the Detached flag.
func RunService(name string, service Service) (*exec.Cmd, error) {
	dir := ExpandPath(service.Path)

	if service.Detached {
		return runDetached(name, service, dir)
	}

	cmdParts := append([]string{service.Command}, service.Args...)
	return openInNewTerminal(name, cmdParts, dir, service.Vars)
}

// runDetached runs the service as a silent background process.
// If a port is set and already in use, it returns an error to prevent blindly
// connecting to an unrelated or zombie process.
func runDetached(name string, service Service, dir string) (*exec.Cmd, error) {
	if service.Port > 0 && isPortInUse(service.Port) {
		return nil, fmt.Errorf("port %d is already in use; cannot safely start service %q", service.Port, name)
	}

	cmd := exec.Command(service.Command, service.Args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Inject environment variables
	env := os.Environ()
	for k, v := range service.Vars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	err := cmd.Start()
	return cmd, err
}

// isPortInUse returns true if a process is already listening on the given TCP port.
func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 300*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// WaitForPort blocks until the given port becomes available, up to a timeout.
func WaitForPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortInUse(port) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}
