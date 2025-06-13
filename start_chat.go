package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
)

// startChat initiates a chat session with a Nomi by name
func startChat(name string) {
	// Ensure the screen is cleared when the program exits
	defer clearScreen()

	// Find the UUID for the given name
	nomiID, err := findNomiByName(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Clear the terminal at the start of the chat
	clearScreen()

	fmt.Printf("\n%s=== Chat Session with %s ===%s\n", colorYellow, name, colorReset)
	fmt.Printf("%s• Type your message and press Enter to send\n", colorBlue)
	fmt.Printf("%s• Type 'exit' to end the session\n", colorBlue)
	fmt.Printf("%s• Use arrow keys to navigate within your text%s\n\n", colorBlue, colorReset)

	// Initialize readline with proper terminal settings
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          fmt.Sprintf("%sYou%s: ", colorGreen, colorReset),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		// Disable persistent history (in-memory history still works during the session)
		DisableAutoSaveHistory: true,
	})
	if err != nil {
		fmt.Printf("Error initializing input reader: %v\n", err)
		return
	}
	defer rl.Close()

	// Set auto-completion function if needed later
	// rl.Config.AutoComplete = completer

	for {
		input, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(input) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		// Check for exit command
		if strings.ToLower(strings.TrimSpace(input)) == "exit" {
			fmt.Println("Chat session ended.")
			break
		}

		// Start the spinner
		stopChan := make(chan bool)
		go spinner(stopChan)

		// Send the message using the API client
		chatResponse, err := client.SendMessage(nomiID, input)

		// Stop the spinner
		close(stopChan)
		fmt.Print("\r") // Clear the spinner line

		if err != nil {
			fmt.Println("Error sending message:", err)
			continue
		}

		// Display the reply
		fmt.Printf("%s%s%s: %s\n", colorBlue, name, colorReset, chatResponse.ReplyMessage.Text)
	}
}
