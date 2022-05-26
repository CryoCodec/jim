package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/services"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
	"os"
	"syscall"
)

func requestPWandDecrypt(uiService services.UiService) {
	attempt := 3
	for {
		if attempt == 0 {
			fmt.Printf("No more attempts left, exiting. \n")
		}

		password := readPasswordFromTerminal()
		err := uiService.Decrypt(password)
		if err != nil {
			attempt--
			fmt.Printf("Decryption failed, reason: %s remaining attempts %d \n", err, attempt)
			continue
		}

		return
	}
}

func readPasswordFromTerminal() []byte {
	fmt.Println("Enter master password:")
	bytePassword, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		fmt.Println("Error reading the password from terminal:", err)
	}
	return bytePassword
}

func runPreamble(uiService services.UiService) error {
	for {
		serverState, err := uiService.GetState()
		if err != nil {
			return err
		}

		if serverState.IsReady() {
			return nil
		}

		if serverState.RequiresDecryption() {
			requestPWandDecrypt(uiService)
		}

		if serverState.RequiresConfigFile() {
			if err := uiService.ReloadConfigFile(); err != nil {
				return err
			}
		}
	}
}

func die(s string) {
	fmt.Println(s)
	os.Exit(1)
}

func dief(s string, el ...interface{}) {
	fmt.Printf(s, el...)
	os.Exit(1)
}

func initLogging() {
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true, DisableLevelTruncation: true})
	if VerbosityLevel >= 2 {
		log.SetLevel(log.TraceLevel)
	} else if VerbosityLevel == 1 {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
}
