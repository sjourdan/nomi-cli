package main

import (
	"fmt"
	"math/rand/v2"
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

const maxRoomReplies = 5

var roomUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var joinRoomCmd = &cobra.Command{
	Use:   "join-room <room-name-or-uuid>",
	Short: "Open an existing group chat room",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		if !roomUUIDPattern.MatchString(id) {
			resolvedID, err := resolveRoomUUID(client, id)
			if err != nil {
				return err
			}
			id = resolvedID
		}

		room, err := client.GetRoom(id)
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
		if err := c.SendRoomMessage(roomID, text); err != nil {
			return replyMsg{conversationID: roomID, err: fmt.Errorf("sending room message: %w", err)}
		}
		result := requestRoomReply(c, roomID, speaker, maxRoomReplies-1)
		if result.err != nil {
			result.err = fmt.Errorf("message sent, %w", result.err)
		}
		return result
	}
}

func requestRoomReply(c *NomiClient, roomID string, speaker Nomi, remaining int) replyMsg {
	result := replyMsg{conversationID: roomID, speaker: speaker, roomRepliesRemaining: remaining}
	reply, err := c.RequestRoomReply(roomID, speaker.UUID)
	if err != nil {
		result.err = fmt.Errorf("requesting a reply from %s: %w", speaker.Name, err)
		return result
	}
	result.reply = reply.Text
	return result
}

func nextRoomSpeaker(nomis []Nomi, previousID string) (Nomi, bool) {
	candidates := make([]Nomi, 0, len(nomis))
	for _, nomi := range nomis {
		if nomi.UUID != previousID {
			candidates = append(candidates, nomi)
		}
	}
	if len(candidates) == 0 {
		return Nomi{}, false
	}
	return candidates[rand.IntN(len(candidates))], true
}
