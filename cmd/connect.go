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
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	jim "github.com/CryoCodec/jim/ipc"
	"github.com/CryoCodec/jim/model"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Opens an interactive SSH connection to the Server, whose tag matches the args the closest.",
	Long:  `Opens an interactive SSH connection to the Server, whose tag matches the args the closest. Requires native SSH and SSHPASS available on PATH.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := jim.CreateClient()
		propagationChan := make(chan jim.Message)
		go jim.ReadMessage(client, propagationChan, Verbose)

		isReady := jim.IsServerStatusReady(client, propagationChan)

		if isReady {
			response := jim.GetMatchingServer(strings.Join(args, " "), client, propagationChan)
			client.Close()
			log.Println("Connection: ", response.Connection)
			err := connectToServer(&response.Server)
			if err != nil {
				log.Fatal("Error: ", err.Error())
			}

		} else {
			log.Fatal("Server is not ready, this is likely an implementation error.")
		}
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// connectCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// connectCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func connectToServer(server *model.Server) error {
	if len(server.Password) == 0 {
		cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-p", server.Port, "-t", server.Username+"@"+server.Host, "cd "+server.Dir+"; "+"bash")
		return interactiveConsole(cmd)
	}

	cmd := exec.Command("sshpass", "-e", "ssh", "-o", "StrictHostKeyChecking=no", "-p", server.Port, "-t", server.Username+"@"+server.Host, "cd "+server.Dir+"; "+"bash")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "SSHPASS="+server.Password)
	return interactiveConsole(cmd)
}

func interactiveConsole(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	return err
}
