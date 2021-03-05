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
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CryoCodec/jim/files"
	"github.com/CryoCodec/jim/model"
	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Performes a system check and gives hints on how to make jim operational.",
	Long:  `Performes a system check and gives hints on how to make jim operational.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Checking if the config directory ~/.jim is available")
		jimDir := files.GetJimConfigDir()
		if _, err := os.Stat(jimDir); os.IsNotExist(err) {
			log.Println("Directory does not exist, creating it...")
			err := os.Mkdir(jimDir, 0740)
			if err != nil {
				log.Fatal("Failed to create jim's config directory", jimDir)
			} else {
				log.Println("---> Success")
			}
		} else {
			log.Println("---> Success")
		}

		log.Println("Checking config file location")
		path := os.Getenv("JIM_CONFIG_FILE") // env variable has highest priority
		if path == "" {
			log.Println("The environment variable JIM_CONFIG_FILE is not set, will use default config location ~/.jim/config.json.enc")
			configFile := filepath.Join(jimDir, "config.json.enc")
			if !files.Exists(configFile) {
				log.Println("The config file does not exist. I will create the dummy file config.json for you. Please update the file and use 'jim encrypt' afterwards.")
				dummyFilePath := filepath.Join(jimDir, "config.json")
				dummyValue := model.JimConfigElement{
					Group: "This is just used for display purposes",
					Env:   "This is just used for display purposes",
					Tag:   "This string is used in the connect command",
					Server: model.Server{
						Host:     "Host name of your Server to connect to",
						Port:     "The SSH Port on the remote server",
						Dir:      "The directory you'd like to start after SSH login",
						Username: "The Username used for authentication",
						Password: "The Password used for authentication",
					},
				}
				jimConfig := model.JimConfig([]model.JimConfigElement{dummyValue})
				json, err := jimConfig.Marshal()
				if err != nil {
					log.Fatal("Failed to deserialize json. This is an implementation bug!")
				}

				if files.Exists(dummyFilePath) {
					log.Println(fmt.Sprintf("The destination path %s already exists, overwrite? (y/n)", dummyFilePath))
					reader := bufio.NewReader(os.Stdin)
					yes, _ := reader.ReadString('\n')
					if strings.TrimSpace(yes) == "y" {
						ioutil.WriteFile(dummyFilePath, json, 0740)
					}
				} else {
					ioutil.WriteFile(dummyFilePath, json, 0740)
				}

			}
		} else {
			log.Println(fmt.Sprintf("Environment variable JIM_CONFIG_FILE is set, using the path %s", path))
		}

		commandExists("pgrep")
		commandExists("sshpass")
		commandExists("ssh")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// doctorCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// doctorCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func commandExists(cmd string) {
	log.Println(fmt.Sprintf("Checking if '%s' is available on Path", cmd))
	_, err := exec.LookPath(cmd)
	if err == nil {
		log.Println("---> Success")
	} else {
		log.Println(fmt.Sprintf("Command '%s' could not be found, but is necessary for jim to work. Please install it and make it available on the PATH", cmd))
	}
}
