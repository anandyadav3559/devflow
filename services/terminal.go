package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/anandyadav3559/devflow/internal/config"
)

// terminalDef describes how to build the argument list for a specific
// terminal emulator to open a new window with a given title and command.
type terminalDef struct {
	bin       string
	buildArgs func(title string, cmd []string, dir string, vars map[string]string, logFile string) []string
}

// supportedTerminals lists known terminal emulators in preference order.
// Each entry knows how to keep the window open after the process exits.
var supportedTerminals = []terminalDef{
	{
		bin: "gnome-terminal",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string, logFile string) []string {
			args := []string{"--title=" + title}
			if dir != "" {
				args = append(args, "--working-directory="+dir)
			}
			args = append(args, "--")
			return append(args, keepOpenShell(cmd, "", vars, logFile)...)
		},
	},
	{
		bin: "kgx", // GNOME Console (default on Fedora 40+)
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string, logFile string) []string {
			return append([]string{"--title=" + title, "--"}, keepOpenShell(cmd, dir, vars, logFile)...)
		},
	},
	{
		bin: "kitty",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string, logFile string) []string {
			// --hold keeps the window open natively after the process exits
			args := []string{"--hold", "--title", title}
			if dir != "" {
				args = append(args, "--directory", dir)
			}
			// Kitty doesn't have a simple way to inject env via CLI, so we use shell wrapper
			return append(args, keepOpenShell(cmd, "", vars, logFile)...)
		},
	},
	{
		bin: "alacritty",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string, logFile string) []string {
			args := []string{"--title", title}
			if dir != "" {
				args = append(args, "--working-directory", dir)
			}
			return append(append(args, "-e"), keepOpenShell(cmd, "", vars, logFile)...)
		},
	},
	{
		bin: "xfce4-terminal",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string, logFile string) []string {
			args := []string{"--title=" + title}
			if dir != "" {
				args = append(args, "--working-directory="+dir)
			}
			return append(append(args, "-x"), keepOpenShell(cmd, "", vars, logFile)...)
		},
	},
	{
		bin: "konsole", // KDE
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string, logFile string) []string {
			args := []string{"--title", title}
			if dir != "" {
				args = append(args, "--workdir", dir)
			}
			return append(append(args, "-e"), keepOpenShell(cmd, "", vars, logFile)...)
		},
	},
	{
		bin: "xterm",
		buildArgs: func(title string, cmd []string, dir string, vars map[string]string, logFile string) []string {
			return append([]string{"-title", title, "-e"}, keepOpenShell(cmd, dir, vars, logFile)...)
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
func keepOpenShell(cmd []string, dir string, vars map[string]string, logFile string) []string {
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

	cmdStr := strings.Join(quotedCmd, " ")
	var script string

	if logFile != "" {
		startEcho := fmt.Sprintf(`echo "--- Start Time: $(date +%%Y-%%m-%%dT%%H:%%M:%%S%%z) ---" > %s; `, shellQuote(logFile))
		endEcho := fmt.Sprintf(`; echo "--- End Time: $(date +%%Y-%%m-%%dT%%H:%%M:%%S%%z) ---" >> %s`, shellQuote(logFile))
		
		script = envPrefix + startEcho + "{ " + cmdStr + "; } 2>&1 | tee -a " + shellQuote(logFile) + endEcho +
			`; echo; echo "--- process exited (press Enter to close) ---"; read`
	} else {
		script = envPrefix + cmdStr +
			`; echo; echo "--- process exited (press Enter to close) ---"; read`
	}

	if dir != "" {
		script = fmt.Sprintf("cd %s && ", shellQuote(dir)) + script
	}
	return []string{"sh", "-c", script}
}

// detectTerminal returns the best available terminal emulator.
// Prefers the terminal set in config, then $TERMINAL or $TERM_PROGRAM,
// then walks the supported list and returns the first one installed.
func detectTerminal() *terminalDef {
	pref := config.Current.Terminal
	if pref != "" && pref != "auto" {
		for i, t := range supportedTerminals {
			if t.bin == pref || strings.HasSuffix(pref, "/"+t.bin) {
				return &supportedTerminals[i]
			}
		}
	}

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
func openInNewTerminal(ctx context.Context, title string, cmd []string, dir string, vars map[string]string, logFile string) (*exec.Cmd, func(), error) {
	if runtime.GOOS == "windows" {
		// Native Windows fallback using cmd.exe
		var envParts []string
		for k, v := range vars {
			envParts = append(envParts, fmt.Sprintf("set %s=%s", k, v))
		}
		envPrefix := ""
		if len(envParts) > 0 {
			envPrefix = strings.Join(envParts, " && ") + " && "
		}

		cmdStr := strings.Join(cmd, " ")
		batCommand := envPrefix + cmdStr + " & echo. & echo --- process exited (press any key to close) --- & pause"

		// Use start command. Title must be first quoted arg.
		args := []string{"/c", "start", title}
		if dir != "" {
			args = append(args, "/D", dir)
		}
		args = append(args, "cmd.exe", "/c", batCommand)

		c := exec.CommandContext(ctx, "cmd.exe", args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		fmt.Printf("  → using cmd.exe\n")
		err := c.Start()
		return c, func() {}, err
	}

	if runtime.GOOS == "darwin" {
		// Native macOS fallback using AppleScript and Terminal.app
		shellCmds := keepOpenShell(cmd, dir, vars, logFile)
		// keepOpenShell returns []string{"sh", "-c", "<script>"}
		// we just need the <script> part to pass to osascript
		scriptStr := shellCmds[2]

		// Escape the script for AppleScript string literals (escape \ and ")
		scriptStr = strings.ReplaceAll(scriptStr, `\`, `\\`)
		scriptStr = strings.ReplaceAll(scriptStr, `"`, `\"`)

		appleScript := fmt.Sprintf(`tell application "Terminal" to do script "%s"`, scriptStr)
		c := exec.CommandContext(ctx, "osascript", "-e", appleScript)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		fmt.Printf("  → using Terminal.app (osascript)\n")
		err := c.Start()
		return c, func() {}, err
	}

	t := detectTerminal()
	if t == nil {
		return nil, nil, fmt.Errorf("no terminal emulator found " +
			"(set $TERMINAL, or install: gnome-terminal, kgx, kitty, alacritty, xfce4-terminal, konsole, xterm)")
	}

	args := t.buildArgs(title, cmd, dir, vars, logFile)
	c := exec.CommandContext(ctx, t.bin, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	fmt.Printf("  → using %s\n", t.bin)
	err := c.Start()
	return c, func() {}, err
}

// ExpandPath expands a leading ~ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
