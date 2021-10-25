package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/services"
	"github.com/spf13/cobra"
	"math"
)

var filters []string
var limit int32

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all entries in the configuration file",
	Long:  `Lists all entries in the configuration file`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()

		uiService := services.NewUiService()
		defer uiService.ShutDown()

		// makes sure the server is in the correct state.
		// might ask the user to enter the master password.
		err := runPreamble(uiService)
		if err != nil {
			dief("Received unexpected error: %s", err)
		}

		groups, err := uiService.GetEntries(filters, int(limit))

		if err != nil {
			die(err.Error())
		}

		fmt.Println()
		for _, group := range *groups {
			fmt.Println(group.Title)
			for _, entry := range group.Entries {
				fmt.Printf("%s -> %s\n", entry.Tag, entry.HostInfo)
			}
			fmt.Println()
		}

		if len(*groups) == 0 {
			fmt.Println("Your query did not yield any results.")
		}
	},
}

func init() {
	filterFlagDescription := `Applies filters to the returned list. 
You may filter over all attributes, or be more precise by using one or multiple of these categories: 
- group
- env
- host
- tag

To filter over all attributes use: '-f "Your text of choice"'
To filter a category, prefix the filter value with the category e.g. '-f "env:INT"'. 
Use this flag multiple times to apply multiple filters e.g. '-f "env:INT" -f "tag:DB"'`

	limitFlagDescription := `Limits the amount entries to be printed. 
The result will include the best matched results. 
This flag is only useful if combined filters.`
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringArrayVarP(&filters, "filter", "f", []string{}, filterFlagDescription)
	listCmd.Flags().Int32VarP(&limit, "limit", "l", math.MaxInt32, limitFlagDescription)
}
