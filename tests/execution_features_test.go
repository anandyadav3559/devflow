package tests

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anandyadav3559/devflow/services"
)

func TestExecutionFeatures(t *testing.T) {
	t.Run("VerifyArgsAndVars", func(t *testing.T) {
		// Use a temporary script to capture output safely
		script := `#!/bin/sh
echo "ARG1=$1"
echo "VAR1=$TEST_VAR"`
		scriptPath := "test_script.sh"
		os.WriteFile(scriptPath, []byte(script), 0755)
		defer os.Remove(scriptPath)

		service := services.Service{
			Command:  "./" + scriptPath,
			Args:     []string{"hello-arg"},
			Vars:     map[string]string{"TEST_VAR": "world-var"},
			Detached: true,
		}

		// Since runDetached writes to os.Stdout, we'll capture it
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		_, err := services.RunService("test-svc", service)
		if err != nil {
			t.Fatalf("Failed to run service: %v", err)
		}

		// Wait a bit for the process to run
		time.Sleep(200 * time.Millisecond)
		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "ARG1=hello-arg") {
			t.Errorf("Expected output to contain ARG1=hello-arg, got:\n%s", output)
		}
		if !strings.Contains(output, "VAR1=world-var") {
			t.Errorf("Expected output to contain VAR1=world-var, got:\n%s", output)
		}
	})

	t.Run("VerifyPortDetection", func(t *testing.T) {
		port := 9999
		// Start a listener on the port
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			t.Fatalf("Failed to listen on port %d: %v", port, err)
		}
		defer ln.Close()

		service := services.Service{
			Command:  "echo",
			Args:     []string{"should be skipped"},
			Port:     port,
			Detached: true,
		}

		_, err = services.RunService("port-svc", service)
		if err == nil {
			t.Fatalf("Expected RunService to return error on port conflict, but got nil")
		}

		if !strings.Contains(err.Error(), "already in use") {
			t.Errorf("Expected error to mention 'already in use', got: %v", err)
		}
	})
}
