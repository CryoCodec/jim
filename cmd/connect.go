package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/services"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const VerboseFlag = "-v"

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
		uiService := services.NewUiService()
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
		initLogging()

		uiService := services.NewUiService()
		defer uiService.ShutDown()

		err := runPreamble(uiService)
		if err != nil {
			dief("Received unexpected error: %s", err)
		}

		query := strings.Join(args, " ")
		response, err := uiService.GetMatchingServer(query)

		if err != nil {
			dief("Error: %s", err)
		}

		fmt.Printf("Connecting to %s -> %s \n", response.Tag, response.Server.Dir)
		err = connectToServer(&response.Server)
		if err != nil {
			dief("Error: %s", err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}

func connectToServer(server *domain.Server) error {
	var sshFlags []string
	if VerbosityLevel >= 1 {
		sshFlags = append(sshFlags, VerboseFlag)
	}

	if len(server.Password) == 0 {
		sshArgs := []string{"-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(server.Port), "-t", server.Username + "@" + server.Host, "cd " + server.Dir + "; " + "bash"}
		sshArgs = append(sshFlags, sshArgs...)
		cmd := exec.Command("ssh", sshArgs...)
		return interactiveConsole(cmd)
	}

	sshPassArgs := []string{"-e", "ssh"}
	sshPassArgs = append(sshPassArgs, sshFlags...)
	sshPassArgs = append(sshPassArgs, "-o", "StrictHostKeyChecking=no", "-p", strconv.Itoa(server.Port), "-t", server.Username+"@"+server.Host, "cd "+server.Dir+"; "+"bash")

	cmd := exec.Command("sshpass", sshPassArgs...)
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
