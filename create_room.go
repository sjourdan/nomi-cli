package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newCreateRoomCmd() *cobra.Command {
	var request CreateRoomRequest
	var participants []string
	cmd := &cobra.Command{
		Use:   "create-room <room-name> --nomi <nomi-name> [--nomi <nomi-name> ...]",
		Short: "Create a group chat room",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			request.Name = args[0]
			if strings.TrimSpace(request.Name) == "" {
				return fmt.Errorf("room name is required")
			}
			if err := validateRoomName(request.Name); err != nil {
				return err
			}
			if err := validateRoomNote(request.Note); err != nil {
				return err
			}

			uuids, err := resolveRoomParticipants(client, participants)
			if err != nil {
				return err
			}
			request.NomiUUIDs = uuids

			room, err := client.CreateRoom(request)
			if err != nil {
				return fmt.Errorf("creating room: %w", err)
			}
			displayRoom(cmd.OutOrStdout(), *room)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&participants, "nomi", nil, "Nomi name or UUID (repeat for each participant)")
	cmd.Flags().StringVar(&request.Note, "note", "", "Room note (up to 1000 characters)")
	cmd.Flags().BoolVar(&request.BackchannelingEnabled, "backchanneling", false, "Enable backchanneling in the room")
	return cmd
}
