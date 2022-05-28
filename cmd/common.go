package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/services"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/theckman/yacspin"
	"golang.org/x/term"
	"os"
	"strings"
	"syscall"
	"time"
)

// Create SprintXxx functions to mix strings with other non-colorized strings:
var green = color.New(color.FgGreen).SprintfFunc()
var red = color.New(color.FgRed).SprintfFunc()
var yellow = color.New(color.FgYellow).SprintfFunc()

func requestPWandDecrypt(uiService services.UiService) {
	enabledSpinner := true
	attempt := 3
outer:
	for {
		if attempt == 0 {
			die("No more attempts left, exiting. \n")
		}

		password := readPasswordFromTerminal()
		spinner, err := CreateSpinner()
		if err != nil {
			enabledSpinner = false
		}
		updateSpinnerPrefix(spinner, "Decoding configuration file")

		if enabledSpinner {
			err = spinner.Start()
			log.Debugf("failed to start the spinner, running without fancy graphics")
			enabledSpinner = false
		}
		channel, err := uiService.Decrypt(password)
		if err != nil {
			dief("\n Encountered an unexpected error: %s", err)
		}
		for update := range channel {
			log.Debugf("received decrypt update: %s", update)
			if update.Error != nil {
				dief("Encountered an unexpected error: %s", update.Error)
			}

			if !update.IsSuccess {
				spinner.StopFail()
				switch update.StepType {
				case domain.Decrypt:
					attempt--
					spinner.StopFail()
					fmt.Printf("Decryption failed, reason: %s remaining attempts %d \n", update.Reason, attempt)
					break outer
				case domain.DecodeBase64:
					fallthrough
				case domain.Validate:
					fallthrough
				case domain.Unmarshal:
					fallthrough
				case domain.BuildIndex:
					fmt.Printf("Reason: %s", update.Reason)
				}

				fmt.Printf("Your configuration file seems to be invalid. Please run 'jim validate' for help")
			}

			switch update.StepType {
			case domain.Done:
				spinner.Stop()
				return
			case domain.DecodeBase64:
				spinner.Stop()
				updateSpinnerPrefix(spinner, "Decrypting configuration file")
				spinner.Start()
			case domain.Decrypt:
				spinner.Stop()
				updateSpinnerPrefix(spinner, "Unmarshalling configuration file")
				spinner.Start()
			case domain.Unmarshal:
				spinner.Stop()
				updateSpinnerPrefix(spinner, "Validating configuration file")
				spinner.Start()
			case domain.Validate:
				spinner.Stop()
				updateSpinnerPrefix(spinner, "Building search index")
				spinner.Start()
			case domain.BuildIndex:
				spinner.Stop()
				updateSpinnerPrefix(spinner, "Writing State")
				spinner.Start()
			}
		}
	}
}

func CreateSpinner() (*yacspin.Spinner, error) {
	// build the configuration, each field is documented
	cfg := yacspin.Config{
		Frequency:         100 * time.Millisecond,
		CharSet:           yacspin.CharSets[11],
		Suffix:            " ", // puts a least one space between the animating spinner and the Message
		SuffixAutoColon:   true,
		ColorAll:          true,
		Colors:            []string{"fgYellow"},
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopMessage:       "done",
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
		StopFailMessage:   "failed",
	}

	s, err := yacspin.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to make spinner from struct: %w", err)
	}

	return s, nil
}

func updateSpinnerPrefix(s *yacspin.Spinner, msg string) {
	offsetNr := 40 - len(msg)
	offset := strings.Repeat(" ", offsetNr)
	s.Prefix(fmt.Sprintf("%s %s", msg, offset))
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
