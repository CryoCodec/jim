package cmd

import (
	"github.com/CryoCodec/jim/core/services"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Prints information for given server entry, whose tag matches the args the closest",
	Long:  `Prints information for given server entry, whose tag matches the args the closest`,
	Run: func(cmd *cobra.Command, args []string) {
		uiService := services.NewUiService(Verbose)
		defer uiService.ShutDown()

		err := runPreamble(uiService)
		if err != nil {
			log.Fatalf("Received unexpected error: %s", err)
		}

		query := strings.Join(args, " ")
		response, err := uiService.GetMatchingServer(query)

		if err != nil {
			log.Fatal(err)
		}

		log.Println("Tag:\t", response.Tag)
		log.Println("Host:\t", response.Server.Host)
		log.Println("Directory:\t", response.Server.Dir)
		log.Println("Username:\t", response.Server.Username)
		log.Println("Password:\t", string(response.Server.Password))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
