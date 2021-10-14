package ipc

import (
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	b64 "encoding/base64"
	"encoding/json"

	"github.com/CryoCodec/jim/crypto"
	"github.com/CryoCodec/jim/files"
	"github.com/schollz/closestmatch"

	ipc "github.com/james-barrow/golang-ipc"
)

// CreateServer creates a server instance with the intended configuration for jim.
// It does not yet react on message, call Listen to actually receive messages.
func CreateServer() *ipc.Server {
	config := ipc.ServerConfig{Timeout: 4 * time.Second}
	sc, err := ipc.StartServer("jimssocket", &config)
	if err != nil {
		log.Fatal("Could not start server, reason:", err)
	}
	return sc
}

// Listen runs a server loop, receiving and answering messages.
// This loop never returns
func Listen(server *ipc.Server) {
	f := runSetup()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		f.Close()
		os.Exit(1)
	}()

	resetC, firedC := startTimer()
	readC := readLoop(server)

	state := serverState{isDecrypted: false, encryptedFileContents: nil}

	for {
		select {
		case <-firedC:
			state.isDecrypted = false
			state.jsonConfig = nil
		case m := <-readC:
			log.Println("Received message " + domain.MsgCodeToString[uint16(m.MsgType)])
			switch m.MsgType {
			case domain.ReqLoadFile:
				handleLoadFile(server, &state, m.Data)
			case domain.ReqStatus:
				handleStatusRequest(server, &state)
			case domain.ReqAttemptDecryption:
				handleDecryption(server, &state, m.Data)
			case domain.ReqListEntries:
				handleListRequest(server, &state)
				go func() { resetC <- true }()
			case domain.ReqClosestMatch:
				handleClosestMatch(server, &state, string(m.Data))
				go func() { resetC <- true }()
			case domain.ReqClosestN:
				handleClosest10(server, &state, string(m.Data))
			default:
				log.Println("Received unexpected message of type " + domain.MsgCodeToString[uint16(m.MsgType)] + ": " + string(m.Data))
				answer(server, domain.ResError, []byte(fmt.Sprintf("Received unexpected message of type %s", domain.MsgCodeToString[uint16(m.MsgType)])))
			}
		}
	}
}

type serverState struct {
	isDecrypted           bool
	encryptedFileContents []byte
	jsonConfig            domain.JimConfig
	matcher               *closestmatch.ClosestMatch
}

func readLoop(server *ipc.Server) chan *ipc.Message {
	out := make(chan *ipc.Message)
	go func() {
		for {
			m, err := server.Read()

			if err == nil {
				switch m.MsgType {
				case -1: // status updates
					log.Printf("State update: %s", server.Status())
				case -2: // internal error
					log.Println("Error: " + err.Error())
				default:
					out <- m
				}
			} else {
				// error case, something went terribly wrong
				// try to give a reason, however this message will probably not be received
				server.Write(domain.ResError, []byte(err.Error()))
				log.Fatal("Fatal error:", err.Error())
			}
		}
	}()
	return out
}

func handleStatusRequest(server *ipc.Server, state *serverState) {
	if state.encryptedFileContents == nil {
		answer(server, domain.ResRequireConfigFile, []byte{})
		return
	}

	if state.isDecrypted {
		answer(server, domain.ResReadyToServe, []byte{})
	} else {
		answer(server, domain.ResNeedDecryption, []byte{})
	}
}

func handleDecryption(server *ipc.Server, state *serverState, passphrase []byte) {
	if state.encryptedFileContents == nil {
		answer(server, domain.ResError, []byte("No configuration was loaded."))
		return
	}

	if state.isDecrypted {
		answer(server, domain.ResSuccess, []byte{})
		return
	}

	cipherText, err := b64.StdEncoding.DecodeString(string(state.encryptedFileContents))
	if err != nil {
		answer(server, domain.ResDecryptionFailed, []byte(fmt.Sprintf("Corrupt configuration file, failed at base64 decode. Reason: %s", err.Error())))
		return
	}

	clearText, err := crypto.Decrypt(passphrase, cipherText)
	if err != nil {
		answer(server, domain.ResDecryptionFailed, []byte(fmt.Sprintf("Failed to decrypt the configuration file. Reason: %s", err.Error())))
		return
	}

	parsed, err := domain.UnmarshalJimConfig(clearText)
	if err != nil {
		answer(server, domain.ResJsonDeserializationFailed, []byte(fmt.Sprintf("Failed to unmarshal json config. Reason: %s", err.Error())))
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
	answer(server, domain.ResSuccess, []byte{})
}

func handleLoadFile(server *ipc.Server, state *serverState, payload []byte) {
	path := string(payload)
	if !files.Exists(path) {
		answer(server, domain.ResError, []byte(fmt.Sprintf("File at %s does not exist or is a directory", path)))
		return
	}

	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		answer(server, domain.ResError, []byte(fmt.Sprintf("Could not read file at %s, reason: %s", path, err.Error())))
		return
	}

	state.encryptedFileContents = fileContents
	state.isDecrypted = false

	answer(server, domain.ResSuccess, []byte{})
}

func handleListRequest(server *ipc.Server, state *serverState) {
	if !state.isDecrypted {
		answer(server, domain.ResNeedDecryption, []byte("Need Decryption"))
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
		el := domain.ListResponseElement{Title: k, Content: v}
		message, err := el.Marshal()
		if err != nil {
			answer(server, domain.ResError, []byte("Failed to serialize json content. This is likely an implementation error"))
			return
		}

		answer(server, domain.ResListEntries, []byte(message))
	}

	answer(server, domain.ResSuccess, []byte{})
}

func handleClosestMatch(server *ipc.Server, state *serverState, query string) {
	if !state.isDecrypted {
		answer(server, domain.ResNeedDecryption, []byte("Need Decryption"))
		return
	}

	// first we try to find the exact match. It can be annoying, when similar tags are used
	// and the wrong one is returned
	for _, config := range state.jsonConfig {
		if config.Tag == query {
			connectionString := fmt.Sprintf("%s -> %s", config.Tag, config.Server.Host)
			response := domain.MatchResponse{Connection: connectionString, Server: config.Server}
			payload, err := response.Marshal()
			if err != nil {
				answer(server, domain.ResError, []byte("Failed to deserialize json. This is likely an implementation error"))
				return
			}
			answer(server, domain.ResClosestMatch, payload)
			return
		}
	}

	// now we try to find the closest match
	match := state.matcher.Closest(strings.ToLower(query))

	for _, config := range state.jsonConfig {
		if config.Tag == match {
			connectionString := fmt.Sprintf("%s -> %s", config.Tag, config.Server.Host)
			response := domain.MatchResponse{Connection: connectionString, Server: config.Server}
			payload, err := response.Marshal()
			if err != nil {
				answer(server, domain.ResError, []byte("Failed to deserialize json. This is likely an implementation error"))
				return
			}
			answer(server, domain.ResClosestMatch, payload)
			return
		}
	}

	answer(server, domain.ResNoMatch, []byte{})
}

func handleClosest10(server *ipc.Server, state *serverState, query string) {
	if !state.isDecrypted {
		answer(server, domain.ResNeedDecryption, []byte("Need Decryption"))
		return
	}

	match := state.matcher.ClosestN(strings.ToLower(query), 10)
	bytes, err := json.Marshal(match)
	if err != nil {
		answer(server, domain.ResError, []byte("Failed to deserialize json. This is likely an implementation error"))
		return
	}

	answer(server, domain.ResClosestN, bytes)
}

func runSetup() *os.File {
	jimDir := files.GetJimConfigDir()
	if _, err := os.Stat(jimDir); os.IsNotExist(err) {
		err := os.Mkdir(jimDir, 0740)
		if err != nil {
			log.Fatal("Failed to create jim's config directory", jimDir)
		}
	}

	f, err := os.OpenFile(filepath.Join(jimDir, "jim-server.log"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0740)
	if err != nil {
		log.Fatalf("Failed to open jim's log file: %v", err)
	}

	log.SetOutput(f)
	log.Println("Setup succeeded")
	return f
}

func answer(server *ipc.Server, code int, message []byte) {
	log.Println("Answering with " + domain.MsgCodeToString[uint16(code)])
	err := server.Write(code, message)
	if err != nil {
		log.Fatal("Error: ", err.Error())
	}
}

func startTimer() (chan bool, chan bool) {
	reset := make(chan bool, 1)
	out := make(chan bool, 1)
	duration := 90 * time.Minute
	timer := time.NewTimer(duration)

	go func() {
		for {
			select {
			case <-reset:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(duration) // this assumes, the timer's channel will be reused
			case <-timer.C:
				out <- true // timer fired, require state update
			}
		}
	}()
	return reset, out
}
