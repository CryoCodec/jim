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
	"path/filepath"
	"strings"
	"syscall"

	b64 "encoding/base64"

	"github.com/CryoCodec/jim/crypto"
	"github.com/CryoCodec/jim/files"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

// decryptCmd represents the decrypt command
var decryptCmd = &cobra.Command{
	Use:   "decrypt path/to/file",
	Short: "Decrypts the file at given path, so you may edit your configuration",
	Long:  `Decrypts the file at given path, so you may edit your configuration. The file has to be encrypted by jim and must end with .enc`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatal("Decrypt expects exactly 1 parameter")
		}

		if !files.Exists(args[0]) {
			log.Fatal("The passed file does not exist or is a directory")
		}

		if filepath.Ext(args[0]) != ".enc" {
			log.Fatal("The passed file did not end on .enc. The file must be encrypted with jim.")
		}

		fileContents, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatal("Error reading file: ", err.Error())
		}

		cipherText, err := b64.StdEncoding.DecodeString(string(fileContents))
		if err != nil {
			log.Fatal("Corrupt input file, failed at base64 decode. Reason: ", err.Error())
		}

		log.Println("Enter master password:")
		password, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal("Error reading the password from terminal. Try again.")
		}

		clearText, err := crypto.Decrypt(password, cipherText)
		if err != nil {
			log.Fatal("Failed to decrypt the given content. Reason: ", err.Error())
		}

		destinationPath := strings.TrimSuffix(args[0], ".enc")

		if files.Exists(destinationPath) {
			log.Println(fmt.Sprintf("The destination path %s already exists, overwrite? (y/n)", destinationPath))
			reader := bufio.NewReader(os.Stdin)
			yes, _ := reader.ReadString('\n')
			if strings.TrimSpace(yes) != "y" {
				return
			}
		}

		ioutil.WriteFile(destinationPath, []byte(clearText), 0644)
		log.Println(fmt.Sprintf("Wrote output to %s", destinationPath))
	},
}

func init() {
	rootCmd.AddCommand(decryptCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// decryptCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// decryptCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
