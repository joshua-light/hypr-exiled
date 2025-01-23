package notify

import (
	"fmt"
	"os/exec"
)

type notificationTool struct {
	name         string
	buildCommand func(tool string, title string, message string, nType NotificationType) *exec.Cmd
}

var notificationTools = []notificationTool{
	{
		name: "dunstify",
		buildCommand: func(tool string, title string, message string, nType NotificationType) *exec.Cmd {
			urgency := "normal"
			if nType == Error {
				urgency = "critical"
				title += " Error"
			}
			return exec.Command(tool, "-u", urgency, "-t", "5000", title, message)
		},
	},
	{
		name: "notify-send",
		buildCommand: func(tool string, title string, message string, nType NotificationType) *exec.Cmd {
			urgency := "normal"
			if nType == Error {
				urgency = "critical"
				title += " Error"
			}
			return exec.Command(tool, "-u", urgency, title, message)
		},
	},
	{
		name: "zenity",
		buildCommand: func(tool string, title string, message string, nType NotificationType) *exec.Cmd {
			flag := "--info"
			if nType == Error {
				flag = "--error"
			}
			return exec.Command(tool, flag, "--text", message, "--title", title)
		},
	},
}

func (n *NotifyService) trySystemNotification(title string, message string, nType NotificationType) error {
	for _, tool := range notificationTools {
		if _, err := exec.LookPath(tool.name); err == nil {
			cmd := tool.buildCommand(tool.name, title, message, nType)
			if err := cmd.Run(); err == nil {
				n.log.Debug("Notification sent successfully",
					"tool", tool.name,
					"type", nType)
				return nil
			}
		}
	}
	return fmt.Errorf("no notification tools available")
}
