// WIP Hotkey package for Hyprland
// decided to move towards flags and letting the user define the keybindings on the WM
// different methods of the app
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	// Define the keybinding and command
	mods := "Ctrl" // Modifier key
	key := "L"     // Key to bind
	command := "dunstify 'Hotkey Triggered' 'Ctrl + L was pressed!'"

	// Log the keybinding and command
	log.Printf("Adding keybinding: %s + %s -> %s\n", mods, key, command)

	// Construct the hyprctl command to add the keybinding
	hyprctlCmd := exec.Command("hyprctl", "keyword", "bind", fmt.Sprintf("%s,%s,exec,%s", mods, key, command))

	// Log the exact command being executed
	log.Printf("Executing command: %s\n", hyprctlCmd.String())

	// Run the command
	output, err := hyprctlCmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to add keybinding: %v\nOutput: %s", err, output)
	}

	// Log the output of the command
	log.Printf("Keybinding added successfully! Output: %s\n", output)

	fmt.Println("Keybinding added successfully!")
	fmt.Println("Press Ctrl + L to trigger the hotkey.")

	// Verify the keybinding was added
	log.Println("Verifying keybinding...")
	hyprctlCmd = exec.Command("hyprctl", "binds")
	output, err = hyprctlCmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to list keybindings: %v\nOutput: %s", err, output)
	}

	// Log the current keybindings
	log.Printf("Current keybindings:\n%s\n", output)

	// Set up a channel to listen for termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a termination signal
	<-sigChan

	// Cleanup: Remove the keybinding
	log.Println("Removing keybinding...")
	hyprctlCmd = exec.Command("hyprctl", "keyword", "unbind", fmt.Sprintf("%s,%s", mods, key))

	// Log the exact command being executed
	log.Printf("Executing command: %s\n", hyprctlCmd.String())

	// Run the command
	output, err = hyprctlCmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to remove keybinding: %v\nOutput: %s", err, output)
	}

	// Log the output of the command
	log.Printf("Keybinding removed successfully! Output: %s\n", output)

	fmt.Println("Keybinding removed successfully!")
}
