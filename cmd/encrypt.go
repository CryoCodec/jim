package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"

	b64 "encoding/base64"

	"github.com/CryoCodec/jim/crypto"
	"github.com/CryoCodec/jim/files"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

// encryptCmd represents the encrypt command
var encryptCmd = &cobra.Command{
	Use:   "encrypt path/to/file",
	Short: "Encrypts the file at given path, so it can be used with jim",
	Long: `Encrypts the file at path/to/file with a master password. 
	The file may then be used with jim`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !files.Exists(args[0]) {
			log.Fatal("The passed file does not exist or is a directory")
		}

		fileContents, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatal("Error reading file: ", err.Error())
		}

		log.Println("Enter master password:")
		password, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal("Error reading the password from terminal. Try again.")
		}

		cipherText, err := crypto.Encrypt(password, fileContents)
		if err != nil {
			log.Fatal("Failed to encrypt the given content. Reason: ", err.Error())
		}

		sEnc := b64.StdEncoding.EncodeToString(cipherText)
		destinationPath := args[0] + ".enc"

		if files.Exists(destinationPath) {
			log.Println(fmt.Sprintf("The destination path %s already exists, overwrite? (y/n)", destinationPath))
			reader := bufio.NewReader(os.Stdin)
			yes, _ := reader.ReadString('\n')
			if strings.TrimSpace(yes) != "y" {
				return
			}
		}

		ioutil.WriteFile(destinationPath, []byte(sEnc), 0644)
		log.Println(fmt.Sprintf("Wrote output to %s", destinationPath))
	},
}

func init() {
	rootCmd.AddCommand(encryptCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// encryptCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// encryptCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
