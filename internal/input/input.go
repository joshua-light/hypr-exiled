package input

import (
	"os/exec"
	"time"
)

func ExecuteInput(cmd string) error {
	if err := exec.Command("wtype", "Return").Run(); err != nil {
		return err
	}

	time.Sleep(50 * time.Millisecond)

	if err := exec.Command("wtype", cmd).Run(); err != nil {
		return err
	}

	time.Sleep(50 * time.Millisecond)

	return exec.Command("wtype", "Return").Run()
}
