package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func displayRoom(out io.Writer, room Room) {
	name := room.Name
	if name == "" {
		name = "<empty>"
	}

	fmt.Fprintf(out, "Room: %s\n", name)
	fmt.Fprintf(out, "- UUID: %s\n", room.UUID)
	fmt.Fprintf(out, "- Created: %s\n", room.Created)
	fmt.Fprintf(out, "- Updated: %s\n", room.Updated)
	fmt.Fprintf(out, "- Status: %s\n", room.Status)
	fmt.Fprintf(out, "- Backchanneling: %v\n", room.BackchannelingEnabled)

	if room.Note != "" {
		fmt.Fprintf(out, "- Note: %s\n", room.Note)
	}

	if len(room.Nomis) > 0 {
		fmt.Fprintln(out, "- Nomis:")
		for _, nomi := range room.Nomis {
			fmt.Fprintf(out, "  • %s (%s, %s)\n",
				nomi.Name,
				nomi.Gender,
				nomi.RelationshipType)
		}
	}
}

var listRoomsCmd = &cobra.Command{
	Use:   "list-rooms",
	Short: "List all rooms",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		rooms, err := client.GetRooms()
		if err != nil {
			fmt.Println("Error fetching rooms:", err)
			return
		}

		// Print the Rooms
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Total Rooms: %d\n\n", len(rooms))
		for _, room := range rooms {
			displayRoom(out, room)
			fmt.Fprintln(out)
		}
	},
}
