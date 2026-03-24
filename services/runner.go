package services

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunService starts a service — either detached (background process) or
// in a new terminal window, depending on the Detached flag. The logDir should be relative or absolute.
func RunService(name string, service Service, globalLog bool, runLogDir string) (*exec.Cmd, func(), error) {
	dir := ExpandPath(service.Path)
	
	shouldLog := service.Log || globalLog
	logFile := ""
	if shouldLog {
		if runLogDir == "" {
			os.MkdirAll("log", 0755)
			runLogDir = "log"
		}
		logFile = filepath.Join(runLogDir, name+".log")
		absPath, err := filepath.Abs(logFile)
		if err == nil {
			logFile = absPath
		}
	}

	if service.Detached {
		return runDetached(name, service, dir, logFile)
	}

	cmdParts := append([]string{service.Command}, service.Args...)
	return openInNewTerminal(name, cmdParts, dir, service.Vars, logFile)
}

// runDetached runs the service as a silent background process.
// If a port is set and already in use, it returns an error to prevent blindly
// connecting to an unrelated or zombie process.
func runDetached(name string, service Service, dir string, logFile string) (*exec.Cmd, func(), error) {
	if service.Port > 0 && isPortInUse(service.Port) {
		return nil, nil, fmt.Errorf("port %d is already in use; cannot safely start service %q", service.Port, name)
	}

	cmd := exec.Command(service.Command, service.Args...)
	if dir != "" {
		cmd.Dir = dir
	}
	
	var finalizer func() = func() {}
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return nil, nil, err
		}
		timeStr := time.Now().Format(time.RFC3339)
		f.WriteString(fmt.Sprintf("--- Start Time: %s ---\n", timeStr))

		cmd.Stdout = io.MultiWriter(os.Stdout, f)
		cmd.Stderr = io.MultiWriter(os.Stderr, f)
		
		finalizer = func() {
			f.WriteString(fmt.Sprintf("--- End Time: %s ---\n", time.Now().Format(time.RFC3339)))
			f.Close()
		}
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Inject environment variables
	env := os.Environ()
	for k, v := range service.Vars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	err := cmd.Start()
	return cmd, finalizer, err
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
