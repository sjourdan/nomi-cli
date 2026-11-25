package main

import (
	"fmt"
	"maps"
	"strings"

	"github.com/spf13/cobra"
)

func newEditRoomCmd() *cobra.Command {
	var name, note string
	var backchanneling bool
	var participants []string
	cmd := &cobra.Command{
		Use:   "edit-room <room-name-or-uuid>",
		Short: "Edit the settings or participants of a group chat room",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var request UpdateRoomRequest
			flags := cmd.Flags()
			if flags.Changed("name") {
				if err := validateRoomName(name); err != nil {
					return err
				}
				request.Name = &name
			}
			if flags.Changed("note") {
				if err := validateRoomNote(note); err != nil {
					return err
				}
				request.Note = &note
			}
			if flags.Changed("backchanneling") {
				request.BackchannelingEnabled = &backchanneling
			}
			if flags.Changed("nomi") {
				uuids, err := resolveRoomParticipants(client, participants)
				if err != nil {
					return err
				}
				request.NomiUUIDs = uuids
			}
			if request.Name == nil && request.Note == nil && request.BackchannelingEnabled == nil && request.NomiUUIDs == nil {
				return fmt.Errorf("specify at least one change using --name, --note, --nomi, or --backchanneling")
			}

			id, err := resolveRoomUUID(client, args[0])
			if err != nil {
				return err
			}
			room, err := client.UpdateRoom(id, request)
			if err != nil {
				return fmt.Errorf("editing room: %w", err)
			}
			if request.NomiUUIDs != nil && !matchesRoomParticipants(room.Nomis, request.NomiUUIDs) {
				room, err = client.GetRoom(id)
				if err != nil {
					return fmt.Errorf("verifying room participants after update: %w", err)
				}
				if !matchesRoomParticipants(room.Nomis, request.NomiUUIDs) {
					names := make([]string, 0, len(room.Nomis))
					for _, nomi := range room.Nomis {
						names = append(names, nomi.Name)
					}
					return fmt.Errorf("Nomi API did not apply the requested participant list. Current participants %q. Edit the participants in the Nomi app", names)
				}
			}
			displayRoom(cmd.OutOrStdout(), *room)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New room name (up to 100 characters)")
	cmd.Flags().StringVar(&note, "note", "", "New room note (pass an empty string to clear)")
	cmd.Flags().BoolVar(&backchanneling, "backchanneling", false, "Set room backchanneling (use --backchanneling=false to disable)")
	cmd.Flags().StringArrayVar(&participants, "nomi", nil, "Request a replacement participant list (repeat for each Nomi name or UUID)")
	return cmd
}

func matchesRoomParticipants(nomis []Nomi, uuids []string) bool {
	requested := make(map[string]struct{}, len(uuids))
	for _, uuid := range uuids {
		requested[strings.ToLower(uuid)] = struct{}{}
	}
	actual := make(map[string]struct{}, len(nomis))
	for _, nomi := range nomis {
		actual[strings.ToLower(nomi.UUID)] = struct{}{}
	}
	return maps.Equal(requested, actual)
}
