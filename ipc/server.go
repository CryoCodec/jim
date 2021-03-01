package ipc

import (
	"bytes"
	"fmt"
	"log"

	ipc "github.com/james-barrow/golang-ipc"
)

func CreateServer() *ipc.Server {
	sc, err := ipc.StartServer("jimssocket", nil)
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
			case ReqAttemptDecryption:
				handleDecryption(server, &state, m.Data)
			default:
				log.Println("Received message of type " + fmt.Sprint(m.MsgType) + ": " + string(m.Data))
			}

		} else {
			// error case, just respond with error message
			log.Println("Error:", err.Error())
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

func handleDecryption(server *ipc.Server, state *serverState, passphrase []byte) {
	if state.isDecrypted {
		server.Write(ResSuccess, []byte{})
		return
	}

	if bytes.Compare(passphrase, []byte("decrypt")) == 0 {
		state.isDecrypted = true
		server.Write(ResSuccess, []byte{})
	} else {
		server.Write(ResDecryptionFailed, []byte{})
	}
}
