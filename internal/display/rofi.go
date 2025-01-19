package display

import (
	"fmt"
	"os/exec"
	"strings"

	"poe-helper/internal/input"
	"poe-helper/internal/models"
)

type RofiManager struct {
	entries []models.TradeEntry
}

func NewRofiManager() (*RofiManager, error) {
	// Check for rofi
	if _, err := exec.LookPath("rofi"); err != nil {
		return nil, fmt.Errorf("rofi not found: %w", err)
	}
	return &RofiManager{}, nil
}

func (r *RofiManager) ShowEntries(entries []models.TradeEntry) error {
	r.entries = entries

	// Format entries for rofi
	var lines []string
	for _, entry := range entries {
		line := fmt.Sprintf("[%s] %s: %s",
			entry.Timestamp.Format("15:04:05"),
			entry.TriggerType,
			entry.PlayerName,
		)
		lines = append(lines, line)
	}

	// Create rofi command
	cmd := exec.Command("rofi", "-dmenu",
		"-p", "Trade Requests",
		"-kb-custom-1", "Ctrl+Shift+i",
		"-kb-custom-2", "Ctrl+Shift+t",
		"-kb-custom-3", "Ctrl+Shift+f",
	)

	// Provide input to rofi
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))

	// Get output
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 10: // Ctrl+Shift+i
				return r.handleInvite(string(output))
			case 11: // Ctrl+Shift+t
				return r.handleTrade(string(output))
			case 12: // Ctrl+Shift+f
				return r.handleFinish(string(output))
			}
		}
		return err
	}

	return nil
}

func (r *RofiManager) handleInvite(selection string) error {
	// Find matching entry
	entry := r.findEntryByLine(selection)
	if entry == nil {
		return fmt.Errorf("no matching entry found")
	}

	// Execute invite command
	cmd := fmt.Sprintf("/invite %s", entry.PlayerName)
	return executeCommand(cmd)
}

func (r *RofiManager) handleTrade(selection string) error {
	entry := r.findEntryByLine(selection)
	if entry == nil {
		return fmt.Errorf("no matching entry found")
	}

	cmd := fmt.Sprintf("@trade %s", entry.PlayerName)
	return executeCommand(cmd)
}

func (r *RofiManager) handleFinish(selection string) error {
	entry := r.findEntryByLine(selection)
	if entry == nil {
		return fmt.Errorf("no matching entry found")
	}

	cmd := fmt.Sprintf("@%s thanks!", entry.PlayerName)
	return executeCommand(cmd)
}

func (r *RofiManager) ShowError(message string) error {
	cmd := exec.Command("rofi", "-e", message)
	return cmd.Run()
}

func (r *RofiManager) findEntryByLine(line string) *models.TradeEntry {
	for _, entry := range r.entries {
		entryLine := fmt.Sprintf("[%s] %s: %s",
			entry.Timestamp.Format("15:04:05"),
			entry.TriggerType,
			entry.PlayerName,
		)
		if strings.TrimSpace(line) == entryLine {
			return &entry
		}
	}
	return nil
}

func executeCommand(cmd string) error {
	// You can reuse your existing input.ExecuteInput function here
	return input.ExecuteInput(cmd)
}
