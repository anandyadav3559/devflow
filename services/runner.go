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
func RunService(name string, service Service) error {
	dir := expandPath(service.Path)

	if service.Detached {
		return runDetached(name, service, dir)
	}

	cmdParts := append([]string{service.Command}, service.Args...)
	return openInNewTerminal(name, cmdParts, dir)
}

// runDetached runs the service as a silent background process.
// If a port is set and already in use, the service is considered already
// running and is skipped gracefully.
func runDetached(name string, service Service, dir string) error {
	if service.Port > 0 && isPortInUse(service.Port) {
		fmt.Printf("  ⚠ port %d already in use — %q is likely running, skipping\n", service.Port, name)
		return nil
	}

	cmd := exec.Command(service.Command, service.Args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
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
