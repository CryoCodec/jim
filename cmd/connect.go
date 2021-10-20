package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/services"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Opens an interactive SSH connection to the Server, whose tag matches the args the closest.",
	Long:  `Opens an interactive SSH connection to the Server, whose tag matches the args the closest. Requires native SSH and SSHPASS available on PATH.`,
	Args:  cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, lastParam string) ([]string, cobra.ShellCompDirective) {
		toComplete := lastParam
		if len(args) != 0 {
			toComplete = fmt.Sprintf("%s %s", strings.Join(args, " "), lastParam)
		}
		uiService := services.NewUiService(Verbose)
		defer uiService.ShutDown()

		if uiService.IsServerReady() {
			cobra.CompDebug(fmt.Sprintf("server is open, trying closestN with %s", toComplete), true)
			arr := uiService.MatchClosestN(toComplete)
			cobra.CompDebug(fmt.Sprintf("Got %v", arr), true)
			return arr, cobra.ShellCompDirectiveNoFileComp
		}
		cobra.CompDebug("Server was not ready, returning nil", true)
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		uiService := services.NewUiService(Verbose)
		defer uiService.ShutDown()

		err := runPreamble(uiService)
		if err != nil {
			log.Fatalf("Received unexpected error: %s", err)
		}

		query := strings.Join(args, " ")
		response, err := uiService.GetMatchingServer(query)

		if err != nil {
			log.Fatal(err)
		}

		log.Println("Tag: ", response.Tag)
		err = connectToServer(&response.Server)
		if err != nil {
			log.Fatal("Error: ", err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}

func connectToServer(server *domain.Server) error {
	if len(server.Password) == 0 {
		cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(server.Port), "-t", server.Username+"@"+server.Host, "cd "+server.Dir+"; "+"bash")
		return interactiveConsole(cmd)
	}

	cmd := exec.Command("sshpass", "-e", "ssh", "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(server.Port), "-t", server.Username+"@"+server.Host, "cd "+server.Dir+"; "+"bash")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "SSHPASS="+string(server.Password))
	return interactiveConsole(cmd)
}

func interactiveConsole(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	return err
}
