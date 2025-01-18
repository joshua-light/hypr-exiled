package input

import (
	"github.com/go-vgo/robotgo"
)

// ExecuteInput simulates typing a command and pressing Enter before and after.
func ExecuteInput(cmd string) error {
	// Simulate pressing the Enter key
	robotgo.KeyTap("enter")

	// Type the command
	robotgo.TypeStr(cmd)

	// Simulate pressing the Enter key again
	robotgo.KeyTap("enter")

	return nil
}
