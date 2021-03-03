/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
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
	Run: func(cmd *cobra.Command, args []string) {
		client := jim.CreateClient()
		propagationChan := make(chan jim.Message)
		go jim.ReadMessage(client, propagationChan, Verbose)

		isReady := jim.IsServerStatusReady(client, propagationChan)

		if isReady {
			entries := jim.ListEntries(client, propagationChan)
			sort.Sort(entries)
			for _, entry := range entries {
				fmt.Println(entry.Title)
				sort.Strings(entry.Content)
				for _, val := range entry.Content {
					fmt.Println(val)
				}
				fmt.Println()
			}
		} else {
			log.Fatal("Server is not ready, this is likely an implementation error.")
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
