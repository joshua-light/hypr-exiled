package notify

import (
	"fmt"
	"os/exec"

	"hypr-exiled/pkg/logger"
)

// NotificationType represents the type of notification
type NotificationType int

const DefaultTitle = "Hypr Exiled"

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

// Show displays a notification with the default title
func (n *NotifyService) Show(message string, nType NotificationType) error {
	return n.ShowWithTitle(DefaultTitle, message, nType)
}

// ShowWithTitle displays a notification with a custom title
func (n *NotifyService) ShowWithTitle(title string, message string, nType NotificationType) error {
	// First try configured notification command if available
	if n.notifyCommand != "" {
		if err := n.executeNotifyCommand(title, message, nType); err == nil {
			return nil
		}
		n.log.Warn("Custom notification command failed", "command", n.notifyCommand)
	}

	// Try system notification tools
	if err := n.trySystemNotification(title, message, nType); err == nil {
		return nil
	}

	// If running in terminal, print directly
	if isRunningInTerminal() {
		return n.printToTerminal(title, message, nType)
	}

	// Try to open a terminal
	if err := n.tryTerminalNotification(title, message, nType); err == nil {
		return nil
	}

	// Last resort: log file
	return n.writeToLogFile(title, message, nType)
}

func (n *NotifyService) executeNotifyCommand(title string, message string, nType NotificationType) error {
	n.log.Debug("executingNotifyCommand",
		"notifyCommand", n.notifyCommand,
		"title", title,
		"nType", nType)

	typeStr := "ERROR"
	if nType == Info {
		typeStr = "INFO"
	}

	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("%s '%s' '%s' '%s'",
			n.notifyCommand,
			typeStr,
			title,
			message))
	return cmd.Run()
}
