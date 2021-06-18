package cmd

import (
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	jim "github.com/CryoCodec/jim/ipc"
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
		ipcPort := jim.InitializeClient(Verbose)
		defer ipcPort.Close()

		if ipcPort.IsServerReady() {
			cobra.CompDebug(fmt.Sprintf("server is open, trying closestN with %s", toComplete), true)
			arr := ipcPort.MatchClosestN(toComplete)
			cobra.CompDebug(fmt.Sprintf("Got %v", arr), true)
			return arr, cobra.ShellCompDirectiveNoFileComp
		}
		cobra.CompDebug("Server was not ready, returning nil", true)
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		ipcPort := jim.InitializeClient(Verbose)

		if !ipcPort.MakeServerReady() {
			log.Fatal("Server is not ready. Unless you've seen other error messages on the screen, this is likely an implementation error.")
		}

		response := ipcPort.GetMatchingServer(strings.Join(args, " "))
		ipcPort.Close()
		log.Println("Connection: ", response.Connection)
		err := connectToServer(&response.Server)
		if err != nil {
			log.Fatal("Error: ", err.Error())
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

func connectToServer(server *domain.Server) error {
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
