package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var fullOutput bool // Flag to control output verbosity

var listNomisCmd = &cobra.Command{
	Use:   "list-nomis",
	Short: "List all Nomis",
	Run: func(cmd *cobra.Command, args []string) {
		nomis, err := client.GetNomis()
		if err != nil {
			fmt.Println("Error fetching Nomis:", err)
			return
		}

		// Display the Nomis
		for _, nomi := range nomis {
			if fullOutput {
				// Full output
				fmt.Printf("- ID: %s\n  Name: %s\n  Gender: %s\n  Created: %s\n  Relationship: %s\n\n",
					nomi.UUID, nomi.Name, nomi.Gender, nomi.Created, nomi.RelationshipType)
			} else {
				// Default output (Name and Relationship only)
				fmt.Printf("%s (%s)\n", nomi.Name, nomi.RelationshipType)
			}
		}
	},
}

func init() {
	// Add the --full flag to the list-nomis command
	listNomisCmd.Flags().BoolVarP(&fullOutput, "full", "f", false, "Display full details of each Nomi")
}
