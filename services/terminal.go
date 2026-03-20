package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// terminalDef describes how to build the argument list for a specific
// terminal emulator to open a new window with a given title and command.
type terminalDef struct {
	bin       string
	buildArgs func(title string, cmd []string, dir string, vars map[string]string) []string
}

// supportedTerminals lists known terminal emulators in preference order.
// Each entry knows how to keep the window open after the process exits.
var supportedTerminals = []terminalDef{
	{
		bin: "gnome-terminal",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string) []string {
			args := []string{"--title=" + title}
			if dir != "" {
				args = append(args, "--working-directory="+dir)
			}
			args = append(args, "--")
			return append(args, keepOpenShell(cmd, "", vars)...)
		},
	},
	{
		bin: "kgx", // GNOME Console (default on Fedora 40+)
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string) []string {
			return append([]string{"--title=" + title, "--"}, keepOpenShell(cmd, dir, vars)...)
		},
	},
	{
		bin: "kitty",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string) []string {
			// --hold keeps the window open natively after the process exits
			args := []string{"--hold", "--title", title}
			if dir != "" {
				args = append(args, "--directory", dir)
			}
			// Kitty doesn't have a simple way to inject env via CLI, so we use shell wrapper
			return append(args, keepOpenShell(cmd, "", vars)...)
		},
	},
	{
		bin: "alacritty",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string) []string {
			args := []string{"--title", title}
			if dir != "" {
				args = append(args, "--working-directory", dir)
			}
			return append(append(args, "-e"), keepOpenShell(cmd, "", vars)...)
		},
	},
	{
		bin: "xfce4-terminal",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string) []string {
			args := []string{"--title=" + title}
			if dir != "" {
				args = append(args, "--working-directory="+dir)
			}
			return append(append(args, "-x"), keepOpenShell(cmd, "", vars)...)
		},
	},
	{
		bin: "konsole", // KDE
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string) []string {
			args := []string{"--title", title}
			if dir != "" {
				args = append(args, "--workdir", dir)
			}
			return append(append(args, "-e"), keepOpenShell(cmd, "", vars)...)
		},
	},
	{
		bin: "xterm",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string) []string {
			return append([]string{"-title", title, "-e"}, keepOpenShell(cmd, dir, vars)...)
		},
	},
}

// shellQuote safely quotes a string for Unix shell execution
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\r$&*(){}[]|;'<>?`~\\\"") {
		return s
	}
	return fmt.Sprintf("%q", s)
}

// keepOpenShell wraps a command in a shell that waits for Enter after the
// process exits, keeping the terminal window visible for the user to read output.
func keepOpenShell(cmd []string, dir string, vars map[string]string) []string {
	var envParts []string
	for k, v := range vars {
		envParts = append(envParts, fmt.Sprintf("export %s=%s", k, shellQuote(v)))
	}
	envPrefix := ""
	if len(envParts) > 0 {
		envPrefix = strings.Join(envParts, " && ") + " && "
	}

	var quotedCmd []string
	for _, arg := range cmd {
		quotedCmd = append(quotedCmd, shellQuote(arg))
	}

	script := envPrefix + strings.Join(quotedCmd, " ") +
		`; echo; echo "--- process exited (press Enter to close) ---"; read`
	if dir != "" {
		script = fmt.Sprintf("cd %s && ", shellQuote(dir)) + script
	}
	return []string{"sh", "-c", script}
}

// detectTerminal returns the best available terminal emulator.
// Prefers the terminal set in $TERMINAL or $TERM_PROGRAM, then walks the
// supported list and returns the first one installed.
func detectTerminal() *terminalDef {
	for _, envKey := range []string{"TERMINAL", "TERM_PROGRAM"} {
		if bin := os.Getenv(envKey); bin != "" {
			for i, t := range supportedTerminals {
				if t.bin == bin || strings.HasSuffix(bin, "/"+t.bin) {
					return &supportedTerminals[i]
				}
			}
		}
	}

	for i, t := range supportedTerminals {
		if _, err := exec.LookPath(t.bin); err == nil {
			return &supportedTerminals[i]
		}
	}

	return nil
}

// openInNewTerminal launches cmd in a new terminal window with the given title
// and optional working directory.
func openInNewTerminal(title string, cmd []string, dir string, vars map[string]string) (*exec.Cmd, error) {
	t := detectTerminal()
	if t == nil {
		return nil, fmt.Errorf("no terminal emulator found " +
			"(set $TERMINAL, or install: gnome-terminal, kgx, kitty, alacritty, xfce4-terminal, konsole, xterm)")
	}

	args := t.buildArgs(title, cmd, dir, vars)
	c := exec.Command(t.bin, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	fmt.Printf("  → using %s\n", t.bin)
	err := c.Start()
	return c, err
}

// expandPath expands a leading ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
