package ipc

import (
	"fmt"
	"io/ioutil"
	"log"

	b64 "encoding/base64"

	"github.com/CryoCodec/jim/crypto"
	"github.com/CryoCodec/jim/files"
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
	state := serverState{isDecrypted: false, encryptedFileContents: nil}

	for {
		m, err := server.Read()

		if err == nil {
			switch m.MsgType {
			case -1: // status updates
				log.Println(fmt.Sprintf("State update: %s", server.Status()))
			case -2: // internal error
				log.Println("Error: " + err.Error())
			case ReqLoadFile:
				handleLoadFile(server, &state, m.Data)
			case ReqStatus:
				handleStatusRequest(server, &state)
			case ReqAttemptDecryption:
				handleDecryption(server, &state, m.Data)
			default:
				log.Println("Received message of type " + fmt.Sprint(m.MsgType) + ": " + string(m.Data))
			}

		} else {
			// error case, something went terribly wrong
			// try to give a reason, however this message will probably not be received
			server.Write(ResError, []byte(err.Error()))
			log.Fatal("Fatal error:", err.Error())
		}
	}
}

type serverState struct {
	isDecrypted           bool
	encryptedFileContents []byte
	clearText             []byte
}

func handleStatusRequest(server *ipc.Server, state *serverState) {
	if state.encryptedFileContents == nil {
		server.Write(ResRequireConfigFile, []byte{})
		return
	}

	if state.isDecrypted {
		server.Write(ResReadyToServe, []byte{})
	} else {
		server.Write(ResNeedDecryption, []byte{})
	}
}

func handleDecryption(server *ipc.Server, state *serverState, passphrase []byte) {
	if state.encryptedFileContents == nil {
		server.Write(ResError, []byte("No configuration was loaded."))
		return
	}

	if state.isDecrypted {
		server.Write(ResSuccess, []byte{})
		return
	}

	cipherText, err := b64.StdEncoding.DecodeString(string(state.encryptedFileContents))
	if err != nil {
		server.Write(ResDecryptionFailed, []byte(fmt.Sprintf("Corrupt configuration file, failed at base64 decode. Reason: %s", err.Error())))
		return
	}

	clearText, err := crypto.Decrypt(passphrase, cipherText)
	if err != nil {
		server.Write(ResDecryptionFailed, []byte(fmt.Sprintf("Failed to decrypt the configuration file. Reason: %s", err.Error())))
		return
	}

	state.clearText = clearText
	state.isDecrypted = true
	server.Write(ResSuccess, []byte{})
}

func handleLoadFile(server *ipc.Server, state *serverState, payload []byte) {
	path := string(payload)
	if !files.Exists(path) {
		server.Write(ResError, []byte(fmt.Sprintf("File at %s does not exist or is a directory", path)))
		return
	}

	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		server.Write(ResError, []byte(fmt.Sprintf("Could not read file at %s, reason: %s", path, err.Error())))
		return
	}

	state.encryptedFileContents = fileContents
	state.isDecrypted = false

	server.Write(ResSuccess, []byte{})
}
