package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/services"
	"strings"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Prints information for given server entry, whose tag matches the args the closest",
	Long:  `Prints information for given server entry, whose tag matches the args the closest`,
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()

		uiService := services.NewUiService()
		defer uiService.ShutDown()

		err := runPreamble(uiService)
		if err != nil {
			dief("Received unexpected error: %s", err)
		}

		query := strings.Join(args, " ")
		response, err := uiService.GetMatchingServer(query)

		if err != nil {
			die(err.Error())
		}

		fmt.Println("Tag:\t\t", response.Tag)
		fmt.Println("Host:\t\t", response.Server.Host)
		fmt.Println("Directory:\t", response.Server.Dir)
		fmt.Println("Username:\t", response.Server.Username)
		fmt.Println("Password:\t", string(response.Server.Password))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
