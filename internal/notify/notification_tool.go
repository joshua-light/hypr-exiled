package notify

import (
	"fmt"
	"os/exec"
)

type notificationTool struct {
	name         string
	buildCommand func(tool string, message string, nType NotificationType) *exec.Cmd
}

var notificationTools = []notificationTool{
	{
		name: "dunstify",
		buildCommand: func(tool string, message string, nType NotificationType) *exec.Cmd {
			urgency := "normal"
			title := "POE Helper Info"
			if nType == Error {
				urgency = "critical"
				title = "POE Helper Error"
			}
			return exec.Command(tool, "-u", urgency, title, message)
		},
	},
	{
		name: "notify-send",
		buildCommand: func(tool string, message string, nType NotificationType) *exec.Cmd {
			urgency := "normal"
			title := "POE Helper Info"
			if nType == Error {
				urgency = "critical"
				title = "POE Helper Error"
			}
			return exec.Command(tool, "-u", urgency, title, message)
		},
	},
	{
		name: "zenity",
		buildCommand: func(tool string, message string, nType NotificationType) *exec.Cmd {
			flag := "--info"
			if nType == Error {
				flag = "--error"
			}
			return exec.Command(tool, flag, "--text", message)
		},
	},
}

func (n *NotifyService) trySystemNotification(message string, nType NotificationType) error {
	for _, tool := range notificationTools {
		if _, err := exec.LookPath(tool.name); err == nil {
			cmd := tool.buildCommand(tool.name, message, nType)
			if err := cmd.Run(); err == nil {
				if n.log != nil {
					n.log.Debug("Notification sent successfully",
						"tool", tool.name,
						"type", nType)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("no notification tools available")
}
