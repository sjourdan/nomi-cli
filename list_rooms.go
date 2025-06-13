package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func displayRoom(room Room) {
	name := room.Name
	if name == "" {
		name = "<empty>"
	}

	fmt.Printf("Room: %s\n", name)
	fmt.Printf("- UUID: %s\n", room.UUID)
	fmt.Printf("- Created: %s\n", room.Created)
	fmt.Printf("- Updated: %s\n", room.Updated)
	fmt.Printf("- Status: %s\n", room.Status)
	fmt.Printf("- Backchanneling: %v\n", room.BackchannelingEnabled)

	if room.Note != "" {
		fmt.Printf("- Note: %s\n", room.Note)
	}

	if len(room.Nomis) > 0 {
		fmt.Println("- Nomis:")
		for _, nomi := range room.Nomis {
			fmt.Printf("  • %s (%s, %s)\n",
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
		fmt.Printf("Total Rooms: %d\n\n", len(rooms))
		for _, room := range rooms {
			displayRoom(room)
			fmt.Println()
		}
	},
}
