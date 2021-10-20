package server

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"fmt"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/crypto"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/CryoCodec/jim/files"
	pb "github.com/CryoCodec/jim/internal/proto"
	"github.com/schollz/closestmatch"
)

type JimServiceImpl struct {
	readChannel  chan readOp
	writeChannel chan writeOp
}

// CreateJimService creates a new grpc server instance
func CreateJimService() pb.JimServer {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received SIGTERM, exiting")
		os.Exit(1)
	}()

	setupLogging()
	readChannel, writeChannel := initializeStateManager()
	return JimServiceImpl{readChannel: readChannel, writeChannel: writeChannel}
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
					read.resp <- state.jsonConfig
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
		jsonConfig:            nil,
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

	parsed, err := domain.UnmarshalJimConfig(clearText)
	if err != nil {
		return &pb.DecryptReply{
			ResponseType: pb.ResponseType_FAILURE,
			Reason:       fmt.Sprintf("Failed to unmarshal json config. Reason: %s", err.Error()),
		}, nil
	}

	var dict []string
	for _, config := range parsed {
		dict = append(dict, config.Tag)
	}

	bagSize := []int{2, 3, 4, 5}
	newState := &serverState{
		isDecrypted:           true,
		encryptedFileContents: state.encryptedFileContents,
		jsonConfig:            parsed,
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
	for _, config := range state.jsonConfig {
		if config.Tag == request.Query {
			pbServer, err := toPbServer(config.Server)
			if err != nil {
				return nil, err
			}
			connectionString := fmt.Sprintf("%s -> %s", config.Tag, config.Server.Host)
			return &pb.MatchReply{Tag: connectionString, Server: pbServer}, nil
		}
	}

	// now we try to find the closest match
	match := state.matcher.Closest(strings.ToLower(request.Query))

	for _, config := range state.jsonConfig {
		if config.Tag == match {
			pbServer, err := toPbServer(config.Server)
			if err != nil {
				return nil, err
			}
			connectionString := fmt.Sprintf("%s -> %s", config.Tag, config.Server.Host)
			return &pb.MatchReply{Tag: connectionString, Server: pbServer}, nil
		}
	}

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
	for _, config := range state.jsonConfig {
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

	return &pb.ListReply{Groups: groups}, nil
}

func (j JimServiceImpl) readState() serverState {
	resp := make(chan interface{})
	j.readChannel <- readOp{opType: ReadServerState, resp: resp}
	val := <-resp
	return val.(serverState)
}

func (j JimServiceImpl) readContent() domain.JimConfig {
	resp := make(chan interface{})
	j.readChannel <- readOp{opType: ReadContent, resp: resp}
	val := <-resp
	return val.(domain.JimConfig)
}

type serverState struct {
	isDecrypted           bool
	encryptedFileContents []byte
	jsonConfig            domain.JimConfig
	matcher               *closestmatch.ClosestMatch
}

// TODO use timer
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

func toPbServer(domainServer domain.JimConfigEntry) (*pb.Server, error) {
	port, err := strconv.Atoi(domainServer.Port)
	if err != nil {
		return nil, err
	}

	return &pb.Server{
		Info:     &pb.PublicServerInfo{Host: domainServer.Host, Directory: domainServer.Dir},
		Port:     int32(port),
		Username: domainServer.Username,
		Password: []byte(domainServer.Password),
	}, nil
}
