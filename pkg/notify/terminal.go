package notify

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (n *NotifyService) tryTerminalNotification(title string, message string, nType NotificationType) error {
	terminal := getSystemTerminal()
	if terminal == "" {
		return fmt.Errorf("no terminal found")
	}

	colorCode := "\\e[32m" // Green for info
	prefix := fmt.Sprintf("%s - %s", title, "Info")
	if nType == Error {
		colorCode = "\\e[31m" // Red for error
		prefix = fmt.Sprintf("%s - %s", title, "Error")
	}

	displayMsg := fmt.Sprintf("echo -e '%s%s:\\e[0m %s\nPress any key to continue...'",
		colorCode, prefix, message)

	var cmd *exec.Cmd
	switch terminal {
	case "gnome-terminal", "xfce4-terminal":
		cmd = exec.Command(terminal, "--", "bash", "-c", displayMsg+"; read -n 1")
	case "konsole":
		cmd = exec.Command(terminal, "-e", "bash", "-c", displayMsg+"; read -n 1")
	default:
		cmd = exec.Command(terminal, "-e", "bash", "-c", displayMsg+"; read -n 1")
	}

	if err := cmd.Run(); err == nil {
		if n.log != nil {
			n.log.Debug("Terminal notification sent",
				"terminal", terminal,
				"type", nType)
		}
		return nil
	}
	return fmt.Errorf("failed to show terminal notification")
}

func (n *NotifyService) writeToLogFile(title string, message string, nType NotificationType) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	typeStr := "INFO"
	if nType == Error {
		typeStr = "ERROR"
	}

	logPath := fmt.Sprintf("%s/.poe-helper-notifications.log", homeDir)
	logMessage := fmt.Sprintf("[%s] %s - %s: %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		title,
		typeStr,
		message)

	if err := os.WriteFile(logPath, []byte(logMessage), 0644); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	if n.log != nil {
		n.log.Debug("Notification written to log file",
			"path", logPath,
			"type", nType)
	}
	return nil
}

func (n *NotifyService) printToTerminal(title string, message string, nType NotificationType) error {
	var colorCode string
	var prefix string

	switch nType {
	case Error:
		colorCode = "\x1b[31m" // Red
		prefix = fmt.Sprintf("%s - Error", title)
	case Info:
		colorCode = "\x1b[32m" // Green
		prefix = fmt.Sprintf("%s - Info", title)
	}

	fmt.Fprintf(os.Stderr, "%s%s: %s\x1b[0m\n", colorCode, prefix, message)
	return nil
}

func isRunningInTerminal() bool {
	// Check if stderr is connected to a terminal
	fileInfo, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// getSystemTerminal tries to find the user's terminal by checking:
// 1. Common terminal environment variables
// 2. $TERM environment variable
// 3. Common terminals list
func getSystemTerminal() string {
	// Check common terminal environment variables
	terminalVars := []string{
		"TERMINAL",              // Common user-set variable
		"TERMCMD",               // Used by some window managers
		"TERM_PROGRAM",          // Used by some terminals
		"TERMINAL_EMULATOR",     // Used by some desktop environments
		"DEFAULT_TERMINAL",      // Sometimes used as a fallback
		"TERMINFO",              // Can indicate terminal type
		"COLORTERM",             // Often set to terminal name
		"KONSOLE_DBUS_SESSION",  // KDE specific
		"GNOME_TERMINAL_SCREEN", // GNOME specific
		"ALACRITTY_LOG",         // Alacritty specific
		"KITTY_WINDOW_ID",       // Kitty specific
	}

	for _, envVar := range terminalVars {
		if terminal := os.Getenv(envVar); terminal != "" {
			// Extract actual terminal name from path if needed
			termName := strings.Split(terminal, "/")
			terminal = termName[len(termName)-1]

			// Remove any arguments or parameters
			terminal = strings.Split(terminal, " ")[0]

			if _, err := exec.LookPath(terminal); err == nil {
				return terminal
			}
		}
	}

	// Check $TERM (but only if it contains an actual terminal name)
	if term := os.Getenv("TERM"); term != "" {
		termName := strings.Split(term, "-")[0]
		if _, err := exec.LookPath(termName); err == nil {
			return termName
		}
	}

	// Fallback to common terminals
	commonTerminals := []string{
		"x-terminal-emulator", // Debian/Ubuntu default
		"gnome-terminal",
		"konsole",
		"xfce4-terminal",
		"alacritty",
		"kitty",
		"terminator",
		"urxvt",
		"rxvt",
		"st",   // Simple Terminal
		"foot", // Wayland terminal
		"wezterm",
		"xterm",
	}

	for _, term := range commonTerminals {
		if _, err := exec.LookPath(term); err == nil {
			return term
		}
	}

	return ""
}
