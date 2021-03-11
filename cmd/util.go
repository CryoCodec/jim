package cmd

import (
	"log"

	jim "github.com/CryoCodec/jim/ipc"
	ipc "github.com/james-barrow/golang-ipc"
)

// ensureServerStatusIsReady is a helper that checks the server status and takes measures to get to ready state
// - if the server is ready, it just returns
// - if the server is not ready, it will try to load the config file and decrypt it
// - if the server cannot reach the ready state, this function will bail out.
func ensureServerStatusIsReady(client *ipc.Client, propagationChan chan jim.Message) {
	if !jim.IsServerStatusReady(client, propagationChan) {
		log.Fatal("Server is not ready. Unless you've seen other error messages on the screen, this is likely an implementation error.")
	}
}
