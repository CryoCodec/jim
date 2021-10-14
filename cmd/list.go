package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/services"
	"github.com/spf13/cobra"
	"log"
	"sort"
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

		entries, err := uiService.GetEntries()
		serviceError, ok := err.(domain.ServiceError)

		if ok && serviceError.IsPasswordRequired() {
			requestPWandDecrypt(uiService)
			entries, err = uiService.GetEntries()
		}

		if err != nil {
			log.Fatal(err)
		}

		for _, entry := range entries {
			fmt.Println(entry.Title)
			sort.Strings(entry.Content)
			for _, val := range entry.Content {
				fmt.Println(val)
			}
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
