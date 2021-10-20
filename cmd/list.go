package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/services"
	"github.com/spf13/cobra"
	"log"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all entries in the configuration file",
	Long:  `Lists all entries in the configuration file`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		uiService := services.NewUiService(Verbose)
		defer uiService.ShutDown()

		// makes sure the server is in the correct state.
		// might ask the user to enter the master password.
		err := runPreamble(uiService)
		if err != nil {
			log.Fatalf("Received unexpected error: %s", err)
		}

		groups, err := uiService.GetEntries()

		if err != nil {
			log.Fatal(err)
		}

		for _, group := range *groups {
			fmt.Println(group.Title)
			for _, entry := range group.Entries {
				fmt.Printf("%s -> %s\n", entry.Tag, entry.HostInfo)
			}
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
