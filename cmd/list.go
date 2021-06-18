package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"log"
	"sort"

	jim "github.com/CryoCodec/jim/ipc"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all entries in the configuration file",
	Long:  `Lists all entries in the configuration file`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ipcPort := jim.InitializeClient(Verbose)
		defer ipcPort.Close()

		if !ipcPort.MakeServerReady() {
			log.Fatal("Server is not ready. Unless you've seen other error messages on the screen, this is likely an implementation error.")
		}

		var entries domain.ListResponse
		c := ipcPort.ListEntries()
		for el := range c {
			entries = append(entries, el)
		}
		sort.Sort(entries)
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
