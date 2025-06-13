package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var getNomiCmd = &cobra.Command{
	Use:   "get-nomi [id]",
	Short: "Get details of a specific Nomi",
	Args:  cobra.ExactArgs(1), // Ensure exactly one argument is passed (the Nomi ID)
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		nomi, err := client.GetNomi(id)
		if err != nil {
			fmt.Println("Error fetching Nomi:", err)
			return
		}

		// Print the Nomi details
		fmt.Println("Nomi Details:")
		fmt.Printf("- ID: %s\n- Name: %s\n- Gender: %s\n- Created: %s\n- Relationship Type: %s\n",
			nomi.UUID, nomi.Name, nomi.Gender, nomi.Created, nomi.RelationshipType)
	},
}
