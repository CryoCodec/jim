package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	jim "github.com/CryoCodec/jim/ipc"
)

// passwordCmd represents the get-password command
var passwordCmd = &cobra.Command{
	Use:   "password <server-id-string>",
	Short: "Returns the password of the selected server.",
	Long:  `Returns the password of the selected server.`,
	Args:  cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, lastParam string) ([]string, cobra.ShellCompDirective) {
		toComplete := lastParam
		if len(args) != 0 {
			toComplete = fmt.Sprintf("%s %s", strings.Join(args, " "), lastParam)
		}
		client := jim.CreateClient()
		defer client.Close()
		propagationChan := jim.StartReceiving(client, Verbose)

		if jim.IsServerStatusReady(client, propagationChan) {
			cobra.CompDebug(fmt.Sprintf("server is open, trying closestN with %s", toComplete), true)
			arr := jim.MatchClosestN(client, propagationChan, toComplete)
			cobra.CompDebug(fmt.Sprintf("Got %v", arr), true)
			return arr, cobra.ShellCompDirectiveNoFileComp
		}
		cobra.CompDebug("Server was not ready, returning nil", true)
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		client := jim.CreateClient()
		propagationChan := jim.StartReceiving(client, Verbose)

		ensureServerStatusIsReady(client, propagationChan)

		response := jim.GetMatchingServer(strings.Join(args, " "), client, propagationChan)
		client.Close()
		fmt.Println(response.Server.Password)
	},
}

func init() {
	rootCmd.AddCommand(passwordCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// connectCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// connectCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
