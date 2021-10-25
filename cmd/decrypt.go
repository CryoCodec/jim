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
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()

		if !files.Exists(args[0]) {
			die("The passed file does not exist or is a directory")
		}

		if filepath.Ext(args[0]) != ".enc" {
			die("The passed file did not end on .enc. The file must be encrypted with jim.")
		}

		fileContents, err := ioutil.ReadFile(args[0])
		if err != nil {
			dief("Error reading file: ", err)
		}

		cipherText, err := b64.StdEncoding.DecodeString(string(fileContents))
		if err != nil {
			dief("Corrupt input file, failed at base64 decode. Reason: ", err)
		}

		log.Println("Enter master password:")
		password, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			die("Error reading the password from terminal. Try again.")

		}

		clearText, err := crypto.Decrypt(password, cipherText)
		if err != nil {
			dief("Failed to decrypt the given content. Reason: ", err)
		}

		destinationPath := strings.TrimSuffix(args[0], ".enc")

		if files.Exists(destinationPath) {
			fmt.Printf("The destination path %s already exists, overwrite? (y/n) \n", destinationPath)
			reader := bufio.NewReader(os.Stdin)
			yes, _ := reader.ReadString('\n')
			if strings.TrimSpace(yes) != "y" {
				return
			}
		}

		err = ioutil.WriteFile(destinationPath, clearText, 0644)
		if err != nil {
			dief("Failed to write to %s: ", err)
		}
		fmt.Printf("Wrote output to %s", destinationPath)
	},
}

func init() {
	rootCmd.AddCommand(decryptCmd)
}
