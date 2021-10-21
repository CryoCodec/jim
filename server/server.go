package server

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	configuration "github.com/CryoCodec/jim/config"
	"github.com/CryoCodec/jim/crypto"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CryoCodec/jim/files"
	pb "github.com/CryoCodec/jim/internal/proto"
	"github.com/schollz/closestmatch"
)

type JimServiceImpl struct {
	readChannel       chan readOp
	writeChannel      chan writeOp
	timerResetChannel chan interface{}
}

// CreateJimService creates a new grpc server instance
func CreateJimService() pb.JimServer {
	setupLogging()
	readChannel, writeChannel := initializeStateManager()
	timerResetChannel := startTimer(writeChannel)
	return JimServiceImpl{readChannel: readChannel, writeChannel: writeChannel, timerResetChannel: timerResetChannel}
}

func setupLogging() {
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
}

type readOp struct {
	opType opType
	resp   chan interface{}
}
type opType int

const (
	ReadServerState = iota
	ReadContent
)

type writeOp struct {
	newState *serverState
}

// initializeStateManager initializes the state governing coroutine.
// Returns two channels to submit read and write Ops.
func initializeStateManager() (chan readOp, chan writeOp) {
	reads := make(chan readOp)
	writes := make(chan writeOp)

	go func() {
		state := serverState{isDecrypted: false, encryptedFileContents: nil}
		for {
			select {
			case read := <-reads:
				switch read.opType {
				case ReadServerState:
					read.resp <- state

				case ReadContent:
					read.resp <- state.config
				}
			case write := <-writes:
				state = *write.newState
			}
		}
	}()

	return reads, writes
}

func (j JimServiceImpl) GetState(ctx context.Context, request *pb.StateRequest) (*pb.StateReply, error) {
	state := j.readState()

	if state.encryptedFileContents == nil {
		return &pb.StateReply{State: pb.StateReply_CONFIG_FILE_REQUIRED}, nil
	}

	if state.isDecrypted {
		return &pb.StateReply{State: pb.StateReply_READY}, nil
	}

	return &pb.StateReply{State: pb.StateReply_DECRYPTION_REQUIRED}, nil
}

func (j JimServiceImpl) LoadConfigFile(ctx context.Context, request *pb.LoadRequest) (*pb.LoadReply, error) {
	path := request.Destination
	if !files.Exists(path) {
		return &pb.LoadReply{
			ResponseType: pb.ResponseType_FAILURE,
			Reason:       fmt.Sprintf("Failed to load config file from %s", path),
		}, nil
	}

	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return &pb.LoadReply{
			ResponseType: pb.ResponseType_FAILURE,
			Reason:       fmt.Sprintf("Could not read file at %s, reason: %s", path, err.Error()),
		}, nil
	}

	newState := &serverState{
		isDecrypted:           false,
		encryptedFileContents: fileContents,
		config:                nil,
		matcher:               nil,
	}

	j.writeChannel <- writeOp{newState: newState}
	return &pb.LoadReply{
		ResponseType: pb.ResponseType_SUCCESS,
		Reason:       "",
	}, nil
}

func (j JimServiceImpl) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptReply, error) {
	state := j.readState()
	if state.encryptedFileContents == nil {
		return &pb.DecryptReply{
			ResponseType: pb.ResponseType_FAILURE,
			Reason:       "No configuration was loaded.",
		}, nil
	}

	if state.isDecrypted {
		return &pb.DecryptReply{
			ResponseType: pb.ResponseType_SUCCESS,
			Reason:       "Already in decrypted state.",
		}, nil
	}

	cipherText, err := b64.StdEncoding.DecodeString(string(state.encryptedFileContents))
	if err != nil {
		return &pb.DecryptReply{
			ResponseType: pb.ResponseType_FAILURE,
			Reason:       fmt.Sprintf("Corrupt configuration file, failed at base64 decode. Reason: %s", err.Error()),
		}, nil
	}

	clearText, err := crypto.Decrypt(request.Password, cipherText)
	if err != nil {
		return &pb.DecryptReply{
			ResponseType: pb.ResponseType_FAILURE,
			Reason:       fmt.Sprintf("Failed to decrypt the configuration file. Reason: %s", err.Error()),
		}, nil
	}

	parsed, err := configuration.UnmarshalJimConfig(clearText)
	if err != nil {
		return &pb.DecryptReply{
			ResponseType: pb.ResponseType_FAILURE,
			Reason:       fmt.Sprintf("Failed to unmarshal json config. Reason: %s", err.Error()),
		}, nil
	}

	var dict []string
	for _, configEntry := range parsed {
		dict = append(dict, configEntry.Tag)
	}

	resultConfig, err := toServerConfig(&parsed)
	if err != nil {
		return nil, err
	}

	bagSize := []int{2, 3, 4, 5}
	newState := &serverState{
		isDecrypted:           true,
		encryptedFileContents: state.encryptedFileContents,
		config:                resultConfig,
		matcher:               closestmatch.New(dict, bagSize),
	}

	j.writeChannel <- writeOp{newState: newState}

	return &pb.DecryptReply{
		ResponseType: pb.ResponseType_SUCCESS,
		Reason:       "Decrypted config file successfully.",
	}, nil
}

func (j JimServiceImpl) Match(ctx context.Context, request *pb.MatchRequest) (*pb.MatchReply, error) {
	state := j.readState()
	if !state.isDecrypted {
		return nil, errors.New("wrong state, requires decryption")
	}

	// first we try to find the exact match. It can be annoying, when similar tags are used
	// and the wrong one is returned
	for _, config := range *state.config {
		if config.Tag == request.Query {
			pbServer := toPbServer(config.Server)
			connectionString := fmt.Sprintf("%s -> %s", config.Tag, config.Server.Host)
			return &pb.MatchReply{Tag: connectionString, Server: pbServer}, nil
		}
	}

	// now we try to find the closest match
	match := state.matcher.Closest(strings.ToLower(request.Query))

	for _, config := range *state.config {
		if config.Tag == match {
			pbServer := toPbServer(config.Server)
			connectionString := fmt.Sprintf("%s -> %s", config.Tag, config.Server.Host)
			return &pb.MatchReply{Tag: connectionString, Server: pbServer}, nil
		}
	}

	j.timerResetChannel <- true // resets the timer
	return nil, errors.New("nothing matched the query")
}

func (j JimServiceImpl) MatchN(ctx context.Context, request *pb.MatchNRequest) (*pb.MatchNReply, error) {
	state := j.readState()
	if !state.isDecrypted {
		return nil, errors.New("wrong state, requires decryption")
	}

	match := state.matcher.ClosestN(strings.ToLower(request.Query), int(request.NumberOfResults))
	return &pb.MatchNReply{Tags: match}, nil
}

func (j JimServiceImpl) List(ctx context.Context, request *pb.ListRequest) (*pb.ListReply, error) {
	state := j.readState()
	if !state.isDecrypted {
		return nil, errors.New("wrong state, requires decryption")
	}

	groupings := make(map[string][]*pb.GroupEntry)
	for _, config := range *state.config {
		title := fmt.Sprintf("%s - %s", config.Group, config.Env)
		value := &pb.GroupEntry{
			Tag: config.Tag,
			Info: &pb.PublicServerInfo{
				Host:      config.Server.Host,
				Directory: config.Server.Dir,
			},
		}
		valSlice := groupings[title]
		valSlice = append(valSlice, value)
		groupings[title] = valSlice
	}

	var groups []*pb.Group
	for title, entries := range groupings {
		groups = append(groups, &pb.Group{
			Title:   title,
			Entries: entries,
		})
	}

	j.timerResetChannel <- true // resets the timer
	return &pb.ListReply{Groups: groups}, nil
}

func (j JimServiceImpl) readState() serverState {
	resp := make(chan interface{})
	j.readChannel <- readOp{opType: ReadServerState, resp: resp}
	val := <-resp
	return val.(serverState)
}

func (j JimServiceImpl) readContent() configuration.JimConfig {
	resp := make(chan interface{})
	j.readChannel <- readOp{opType: ReadContent, resp: resp}
	val := <-resp
	return val.(configuration.JimConfig)
}

type serverState struct {
	isDecrypted           bool
	encryptedFileContents []byte
	config                *Config
	matcher               *closestmatch.ClosestMatch
}

// startTimer starts a timer, that will periodically force the server
// to close the encrypted state, if not used. This requires the client to run the preamble
// once again. Returns a channel to reset the timer when written to.
func startTimer(writeChannel chan writeOp) chan interface{} {
	reset := make(chan interface{}, 1)
	duration := 90 * time.Minute
	timer := time.NewTimer(duration)
	resetState := &serverState{isDecrypted: false, encryptedFileContents: nil}

	go func() {
		for {
			select {
			case <-reset:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(duration) // this assumes, the timer's channel will be reused
			case <-timer.C:
				writeChannel <- writeOp{newState: resetState} // timer fired, require state update
				timer.Reset(duration)
			}
		}
	}()
	return reset
}

func toPbServer(domainServer ConfigEntry) *pb.Server {
	return &pb.Server{
		Info:     &pb.PublicServerInfo{Host: domainServer.Host, Directory: domainServer.Dir},
		Port:     int32(domainServer.Port),
		Username: domainServer.Username,
		Password: domainServer.Password,
	}
}

// Config is a type alias for a list of config elements
type Config []ConfigElement

// ConfigElement is the main structure used within the server
type ConfigElement struct {
	Group  string
	Env    string
	Tag    string
	Server ConfigEntry
}

// ConfigEntry holds all the information necessary to connect to a server via ssh
type ConfigEntry struct {
	Host     string
	Dir      string
	Port     int
	Username string
	Password []byte
}

func toServerConfig(jimConfig *configuration.JimConfig) (*Config, error) {
	var result Config
	for _, el := range *jimConfig {
		server := el.Server
		port, err := strconv.Atoi(server.Port)
		if err != nil {
			return nil, errors.Errorf("Encountered invalid port in config file: %s", server.Port)
		}
		newEl := ConfigElement{
			Group: el.Group,
			Env:   el.Env,
			Tag:   el.Tag,
			Server: ConfigEntry{
				Host:     server.Host,
				Dir:      server.Dir,
				Port:     port,
				Username: server.Username,
				Password: []byte(server.Password),
			},
		}
		result = append(result, newEl)
	}
	return &result, nil
}
