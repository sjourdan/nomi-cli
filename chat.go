package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type ChatRequest struct {
	MessageText string `json:"messageText"`
}

type Message struct {
	UUID string `json:"uuid"`
	Text string `json:"text"`
	Sent string `json:"sent"`
}

type ChatResponse struct {
	SentMessage  Message `json:"sentMessage"`
	ReplyMessage Message `json:"replyMessage"`
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorBlue   = "\033[34m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// clearScreen clears the terminal screen and attempts to clear the scrollback buffer.
func clearScreen() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		// First attempt using ANSI escape codes
		fmt.Print("\033[H\033[2J\033[3J\033c")

		// Fallback to tput if available
		cmd := exec.Command("tput", "reset")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

// findNomiByName retrieves the UUID of a Nomi by its name using case-insensitive comparison.
func findNomiByName(name string) (string, error) {
	nomis, err := client.GetNomis()
	if err != nil {
		return "", err
	}

	// Search for the Nomi by name (case-insensitive)
	for _, nomi := range nomis {
		if strings.EqualFold(nomi.Name, name) {
			return nomi.UUID, nil
		}
	}

	return "", fmt.Errorf("no Nomi found with the name: %s", name)
}

// spinner displays a spinning wheel animation while waiting for a response.
func spinner(stopChan chan bool) {
	chars := []string{"-", "\\", "|", "/"} // Simple classic spinner
	for {
		select {
		case <-stopChan:
			return
		default:
			for _, char := range chars {
				select {
				case <-stopChan:
					return
				default:
					fmt.Printf("\r%s%s%s", colorCyan, char, colorReset)
					time.Sleep(100 * time.Millisecond) // Slightly slower rotation
				}
			}
		}
	}
}

var chatCmd = &cobra.Command{
	Use:   "chat [id]",
	Short: "Start a live chat session with a specific Nomi",
	Args:  cobra.ExactArgs(1), // Requires exactly one argument: the Nomi Name
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		startChat(name)
	},
}
