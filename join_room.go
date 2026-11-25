package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var joinRoomCmd = &cobra.Command{
	Use:   "join-room <room-uuid>",
	Short: "Open an existing group chat room",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		room, err := client.GetRoom(args[0])
		if err != nil {
			return fmt.Errorf("loading room: %w", err)
		}
		if len(room.Nomis) == 0 {
			return fmt.Errorf("room %q has no Nomis", room.Name)
		}
		_, err = tea.NewProgram(newRoomChatTUI(*room), tea.WithAltScreen()).Run()
		return err
	},
}

func sendRoomCmd(c *NomiClient, roomID string, speaker Nomi, text string) tea.Cmd {
	return func() tea.Msg {
		result := replyMsg{conversationID: roomID}
		if err := c.SendRoomMessage(roomID, text); err != nil {
			result.err = fmt.Errorf("sending room message: %w", err)
			return result
		}
		reply, err := c.RequestRoomReply(roomID, speaker.UUID)
		if err != nil {
			result.err = fmt.Errorf("message sent, requesting a reply from %s: %w", speaker.Name, err)
			return result
		}
		result.reply = reply.Text
		result.speaker = speaker.Name
		return result
	}
}
