package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteRoomCmd = &cobra.Command{
	Use:   "delete-room <room-name-or-uuid>",
	Short: "Delete a group chat room",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selector := args[0]
		if selector == "" {
			return fmt.Errorf("provide a room name or UUID")
		}
		id, err := resolveRoomUUID(client, selector)
		if err != nil {
			return err
		}
		if err := client.DeleteRoom(id); err != nil {
			return fmt.Errorf("deleting room: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Deleted room %q\n", selector)
		return nil
	},
}
