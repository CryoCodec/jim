package ipc

import (
	"fmt"
	"log"

	ipc "github.com/james-barrow/golang-ipc"
)

func CreateServer() *ipc.Server {
	serverConfig := &ipc.ServerConfig{Encryption: true}
	sc, err := ipc.StartServer("jimssocket", serverConfig)
	if err != nil {
		log.Fatal("Could not start server, reason:", err)
	}
	return sc
}

func Listen(server *ipc.Server) {
	state := serverState{false}

	for {
		m, err := server.Read()

		if err == nil {
			switch m.MsgType {
			case -2: // internal error
				log.Println("Error: " + err.Error())
			case ReqStatus:
				handleStatusRequest(server, &state)
			default:
				log.Println("Received message of type " + fmt.Sprint(m.MsgType) + ": " + string(m.Data))
			}

		} else {
			// error case, just respond with error message
			server.Write(ResError, []byte(err.Error()))
		}
	}
}

type serverState struct {
	isDecrypted bool
}

func handleStatusRequest(server *ipc.Server, state *serverState) {
	if state.isDecrypted {
		server.Write(ResReadyToServe, []byte{})
	} else {
		server.Write(ResNeedDecryption, []byte{})
	}
}
