package ipc

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	b64 "encoding/base64"

	"github.com/CryoCodec/jim/crypto"
	"github.com/CryoCodec/jim/files"
	"github.com/CryoCodec/jim/model"
	"github.com/schollz/closestmatch"

	ipc "github.com/james-barrow/golang-ipc"
)

func CreateServer() *ipc.Server {
	config := ipc.ServerConfig{Timeout: 4 * time.Second}
	sc, err := ipc.StartServer("jimssocket", &config)
	if err != nil {
		log.Fatal("Could not start server, reason:", err)
	}
	return sc
}

func Listen(server *ipc.Server) {
	f := runSetup()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		f.Close()
		os.Exit(1)
	}()

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
				log.Println("Received message " + msgCodeToString[uint16(m.MsgType)])
				handleLoadFile(server, &state, m.Data)
			case ReqStatus:
				log.Println("Received message " + msgCodeToString[uint16(m.MsgType)])
				handleStatusRequest(server, &state)
			case ReqAttemptDecryption:
				log.Println("Received message " + msgCodeToString[uint16(m.MsgType)])
				handleDecryption(server, &state, m.Data)
			case ReqListEntries:
				log.Println("Received message " + msgCodeToString[uint16(m.MsgType)])
				handleListRequest(server, &state)
			case ReqClosestMatch:
				log.Println("Received message " + msgCodeToString[uint16(m.MsgType)])
				handleClosestMatch(server, &state, string(m.Data))
			default:
				log.Println("Received unexpected message of type " + msgCodeToString[uint16(m.MsgType)] + ": " + string(m.Data))
				answer(server, ResError, []byte(fmt.Sprintf("Received unexpected message of type %s", msgCodeToString[uint16(m.MsgType)])))
			}

		} else {
			// error case, something went terribly wrong
			// try to give a reason, however this message will probably not be received
			server.Write(ResError, []byte(err.Error()))
			f.Close()
			log.Fatal("Fatal error:", err.Error())
		}
	}
}

type serverState struct {
	isDecrypted           bool
	encryptedFileContents []byte
	jsonConfig            model.JimConfig
	matcher               *closestmatch.ClosestMatch
}

func handleStatusRequest(server *ipc.Server, state *serverState) {
	if state.encryptedFileContents == nil {
		answer(server, ResRequireConfigFile, []byte{})
		return
	}

	if state.isDecrypted {
		answer(server, ResReadyToServe, []byte{})
	} else {
		answer(server, ResNeedDecryption, []byte{})
	}
}

func handleDecryption(server *ipc.Server, state *serverState, passphrase []byte) {
	if state.encryptedFileContents == nil {
		answer(server, ResError, []byte("No configuration was loaded."))
		return
	}

	if state.isDecrypted {
		answer(server, ResSuccess, []byte{})
		return
	}

	cipherText, err := b64.StdEncoding.DecodeString(string(state.encryptedFileContents))
	if err != nil {
		answer(server, ResDecryptionFailed, []byte(fmt.Sprintf("Corrupt configuration file, failed at base64 decode. Reason: %s", err.Error())))
		return
	}

	clearText, err := crypto.Decrypt(passphrase, cipherText)
	if err != nil {
		answer(server, ResDecryptionFailed, []byte(fmt.Sprintf("Failed to decrypt the configuration file. Reason: %s", err.Error())))
		return
	}

	parsed, err := model.UnmarshalJimConfig(clearText)
	if err != nil {
		answer(server, ResJsonDeserializationFailed, []byte(fmt.Sprintf("Failed to unmarshal json config. Reason: %s", err.Error())))
		return
	}

	var dict []string
	for _, config := range parsed {
		dict = append(dict, config.Tag)
	}

	bagSize := []int{2, 3, 4, 5}
	state.matcher = closestmatch.New(dict, bagSize)
	state.jsonConfig = parsed
	state.isDecrypted = true
	answer(server, ResSuccess, []byte{})
}

func handleLoadFile(server *ipc.Server, state *serverState, payload []byte) {
	path := string(payload)
	if !files.Exists(path) {
		answer(server, ResError, []byte(fmt.Sprintf("File at %s does not exist or is a directory", path)))
		return
	}

	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		answer(server, ResError, []byte(fmt.Sprintf("Could not read file at %s, reason: %s", path, err.Error())))
		return
	}

	state.encryptedFileContents = fileContents
	state.isDecrypted = false

	answer(server, ResSuccess, []byte{})
}

func handleListRequest(server *ipc.Server, state *serverState) {
	if !state.isDecrypted {
		answer(server, ResNeedDecryption, []byte("Need Decryption"))
		return
	}

	groupings := make(map[string][]string)
	for _, config := range state.jsonConfig {
		title := fmt.Sprintf("%s - %s", config.Group, config.Env)
		value := fmt.Sprintf("%s -> %s : %s", config.Tag, config.Server.Host, config.Server.Dir)
		valSlice := groupings[title]
		valSlice = append(valSlice, value)
		groupings[title] = valSlice
	}

	for k, v := range groupings {
		el := model.ListResponseElement{Title: k, Content: v}
		message, err := el.Marshal()
		if err != nil {
			answer(server, ResError, []byte("Failed to serialize json content. This is likely an implementation error"))
			return
		}

		answer(server, ResListEntries, []byte(message))
	}

	answer(server, ResSuccess, []byte{})
}

func handleClosestMatch(server *ipc.Server, state *serverState, query string) {
	if !state.isDecrypted {
		answer(server, ResNeedDecryption, []byte("Need Decryption"))
		return
	}

	// first we try to find the exact match. It can be annoying, when similar tags are used
	// and the wrong one is returned
	for _, config := range state.jsonConfig {
		if config.Tag == query {
			connectionString := fmt.Sprintf("%s -> %s", config.Tag, config.Server.Host)
			response := model.MatchResponse{Connection: connectionString, Server: config.Server}
			payload, err := response.Marshal()
			if err != nil {
				answer(server, ResError, []byte("Failed to deserialize json. This is likely an implementation error"))
				return
			}
			answer(server, ResClosestMatch, payload)
			return
		}
	}

	// now we try to find the closest match
	match := state.matcher.Closest(strings.ToLower(query))

	for _, config := range state.jsonConfig {
		if config.Tag == match {
			connectionString := fmt.Sprintf("%s -> %s", config.Tag, config.Server.Host)
			response := model.MatchResponse{Connection: connectionString, Server: config.Server}
			payload, err := response.Marshal()
			if err != nil {
				answer(server, ResError, []byte("Failed to deserialize json. This is likely an implementation error"))
				return
			}
			answer(server, ResClosestMatch, payload)
			return
		}
	}

	answer(server, ResNoMatch, []byte{})
}

func runSetup() *os.File {
	jimDir := files.GetJimConfigDir()
	if _, err := os.Stat(jimDir); os.IsNotExist(err) {
		err := os.Mkdir(jimDir, 0740)
		if err != nil {
			log.Fatal("Failed to create jim's config directory", jimDir)
		}
	}

	f, err := os.OpenFile(filepath.Join(jimDir, "jim-server.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0740)
	if err != nil {
		log.Fatalf("Failed to open jim's log file: %v", err)
	}

	log.SetOutput(f)
	log.Println("Setup succeeded")
	return f
}

func answer(server *ipc.Server, code int, message []byte) {
	log.Println("Answering with " + msgCodeToString[uint16(code)])
	err := server.Write(code, message)
	if err != nil {
		log.Fatal("Error: ", err.Error())
	}
}
