package main

import (
	"fmt"

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

var chatCmd = &cobra.Command{
	Use:   "chat [name]",
	Short: "Start an interactive chat session with a specific Nomi",
	Args:  cobra.ExactArgs(1), // Requires exactly one argument: the Nomi name
	RunE: func(cmd *cobra.Command, args []string) error {
		nomis, err := client.GetNomis()
		if err != nil {
			return fmt.Errorf("fetching Nomis: %w", err)
		}
		return runTUI(nomis, args[0])
	},
}
