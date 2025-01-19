package notify

import (
	"fmt"
	"os/exec"

	"poe-helper/pkg/logger"
)

// NotificationType represents the type of notification
type NotificationType int

const (
	Error NotificationType = iota
	Info
)

// NotifyService handles system notifications
type NotifyService struct {
	log           *logger.Logger
	notifyCommand string
}

// NewNotifyService creates a new notification service
func NewNotifyService(notifyCommand string, log *logger.Logger) *NotifyService {
	return &NotifyService{
		log:           log,
		notifyCommand: notifyCommand,
	}
}

// Show displays a notification of the specified type
func (n *NotifyService) Show(message string, nType NotificationType) error {
	// First try configured notification command if available
	if n.notifyCommand != "" {
		if err := n.executeNotifyCommand(message, nType); err == nil {
			return nil
		}
		n.log.Warn("Custom notification command failed", "command", n.notifyCommand)
	}

	// Try system notification tools
	if err := n.trySystemNotification(message, nType); err == nil {
		return nil
	}

	// If running in terminal, print directly
	if isRunningInTerminal() {
		return n.printToTerminal(message, nType)
	}

	// Try to open a terminal
	if err := n.tryTerminalNotification(message, nType); err == nil {
		return nil
	}

	// Last resort: log file
	return n.writeToLogFile(message, nType)
}

func (n *NotifyService) executeNotifyCommand(message string, nType NotificationType) error {
	n.log.Debug("executingNotifyhCommand", "notifyCommand", n.notifyCommand,
		"nType", nType)
	typeStr := "ERROR"
	if nType == Info {
		typeStr = "INFO"
	}

	cmd := exec.Command("sh", "-c", fmt.Sprintf("%s '%s' '%s'", n.notifyCommand, typeStr, message))
	return cmd.Run()
}
