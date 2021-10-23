package server

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	configuration "github.com/CryoCodec/jim/config"
	"github.com/CryoCodec/jim/crypto"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/CryoCodec/jim/files"
	pb "github.com/CryoCodec/jim/internal/proto"
	"github.com/blevesearch/bleve/v2"
)

type JimServiceImpl struct {
	readChannel       chan readOp
	writeChannel      chan writeOp
	timerResetChannel chan interface{}
}

// CreateJimService creates a new grpc server instance
func CreateJimService() pb.JimServer {
	defer timeTrack(time.Now(), "setup")
	setupLogging()
	readChannel, writeChannel := initializeStateManager()
	timerResetChannel := startTimer(writeChannel)
	return JimServiceImpl{
		readChannel:       readChannel,
		writeChannel:      writeChannel,
		timerResetChannel: timerResetChannel}
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
	WriteCloseState
	WriteState
)

type writeOp struct {
	opType   opType
	newState *serverState
}

// initializeStateManager initializes the state governing coroutine.
// Returns two channels to submit read and write Ops.
func initializeStateManager() (chan readOp, chan writeOp) {
	reads := make(chan readOp, 3)
	writes := make(chan writeOp, 3)

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
				switch write.opType {
				case WriteCloseState:
					state.isDecrypted = false
					state.config = nil
					state.grouping = nil
					if state.index != nil {
						go func() {
							err := state.index.Close()
							if err != nil {
								log.Printf("Error when closing the index: %s", err)
							}
						}()
					}
					state.index = nil
				case WriteState:
					state = *write.newState
				}

			}
		}
	}()

	return reads, writes
}

func (j JimServiceImpl) GetState(ctx context.Context, request *pb.StateRequest) (*pb.StateReply, error) {
	defer timeTrack(time.Now(), "GetState")

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
	defer timeTrack(time.Now(), "LoadConfigFile")

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
		index:                 nil,
	}

	// close previously opened states. This may be required when this function is used with the 'reload' cmd
	j.writeChannel <- writeOp{newState: newState, opType: WriteCloseState}
	// write new state
	j.writeChannel <- writeOp{newState: newState, opType: WriteState}
	return &pb.LoadReply{
		ResponseType: pb.ResponseType_SUCCESS,
		Reason:       "",
	}, nil
}

func (j JimServiceImpl) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptReply, error) {
	defer timeTrack(time.Now(), "Decrypt")

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

	resultConfig, err := toServerConfig(&parsed)
	if err != nil {
		return nil, err
	}

	// create the bleve index
	type pair struct {
		index bleve.Index
		err   error
	}

	returnChan := make(chan pair)
	go func() {
		hash, err := hashstructure.Hash(resultConfig, hashstructure.FormatV2, nil)
		if err != nil {
			returnChan <- pair{
				index: nil,
				err:   errors.Errorf("Failed to hash the config: %d", hash),
			}
		}

		index, err := createIndex(strconv.Itoa(int(hash)), resultConfig)
		if err != nil {
			returnChan <- pair{
				index: nil,
				err:   err,
			}
		}
		returnChan <- pair{
			index: index,
			err:   nil,
		}
	}()

	// create grouping table for quickly accessing the matched tag
	groupTable := make(map[string]*ConfigElement)
	for _, entry := range *resultConfig {
		copiedEntry := entry // this is required! otherwise & operator always points to the loop variable
		groupTable[entry.Tag] = &copiedEntry
	}

	result := <-returnChan
	if result.err != nil {
		return nil, err
	}

	newState := &serverState{
		isDecrypted:           true,
		encryptedFileContents: state.encryptedFileContents,
		config:                resultConfig,
		index:                 result.index,
		grouping:              groupTable,
	}

	j.writeChannel <- writeOp{newState: newState, opType: WriteState}

	return &pb.DecryptReply{
		ResponseType: pb.ResponseType_SUCCESS,
		Reason:       "Decrypted config file successfully.",
	}, nil
}

func (j JimServiceImpl) Match(ctx context.Context, request *pb.MatchRequest) (*pb.MatchReply, error) {
	defer timeTrack(time.Now(), "Match")

	state := j.readState()
	if !state.isDecrypted {
		return nil, errors.New("wrong state, requires decryption")
	}

	log.Printf("User queried '%s'", request.Query)
	// now we try to find the closest match
	query := bleve.NewMatchQuery(fmt.Sprintf("\"%s\"", request.Query))
	search := bleve.NewSearchRequest(query)
	search.Size = 1
	search.Fields = []string{"tag"}
	searchResults, err := state.index.Search(search)

	if err != nil {
		return nil, errors.Errorf("Encountered an unexpected error during search: %s", err)
	}

	if len(searchResults.Hits) != 0 {
		tag := searchResults.Hits[0].Fields["tag"].(string)
		log.Printf("Query matched '%s'", tag)
		configEl, ok := state.grouping[tag]
		if ok {
			return &pb.MatchReply{
				Tag:    tag,
				Server: toPbServer(configEl.Server),
			}, nil
		}
	}

	j.timerResetChannel <- true // resets the timer
	return nil, errors.New("nothing matched the query")
}

// todo remove, no longer required?
func (j JimServiceImpl) MatchN(ctx context.Context, request *pb.MatchNRequest) (*pb.MatchNReply, error) {

	state := j.readState()
	if !state.isDecrypted {
		return nil, errors.New("wrong state, requires decryption")
	}

	return &pb.MatchNReply{Tags: []string{}}, nil
}

func (j JimServiceImpl) List(ctx context.Context, request *pb.ListRequest) (*pb.ListReply, error) {
	defer timeTrack(time.Now(), "List")

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
	grouping              map[string]*ConfigElement
	index                 bleve.Index
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
				writeChannel <- writeOp{newState: resetState, opType: WriteCloseState} // close old state
				timer.Reset(duration)
			}
		}
	}()
	return reset
}

func toPbServer(domainServer ServerEntry) *pb.Server {
	return &pb.Server{
		Info:     &pb.PublicServerInfo{Host: domainServer.Host, Directory: domainServer.Dir},
		Port:     int32(domainServer.Port),
		Username: domainServer.Credentials.Username,
		Password: domainServer.Credentials.Password,
	}
}

// Config is a type alias for a list of config elements
type Config []ConfigElement

// ConfigElement is the main structure used within the server
type ConfigElement struct {
	Group  string
	Env    string
	Tag    string
	Server ServerEntry
}

func (c ConfigElement) String() string {
	return fmt.Sprintf("ConfigElement{ group=%s, env=%s, tag=%s, host=%s, Dir=%s }", c.Group, c.Env, c.Tag, c.Server.Host, c.Server.Dir)
}

// ServerEntry holds all the information necessary to connect to a server via ssh
type ServerEntry struct {
	Host        string
	Dir         string
	Port        int
	Credentials credentials
}

type credentials struct {
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
			Server: ServerEntry{
				Host: server.Host,
				Dir:  server.Dir,
				Port: port,
				Credentials: credentials{
					Username: server.Username,
					Password: []byte(server.Password),
				},
			},
		}
		result = append(result, newEl)
	}
	return &result, nil
}

type indexDocument struct {
	Group string `json:"group"`
	Env   string `json:"env"`
	Tag   string `json:"tag"`
	Host  string `json:"host"`
}

func (i indexDocument) Type() string {
	return "indexDocument"
}

func buildIndexMapping() *mapping.IndexMappingImpl {
	// a generic reusable mapping for english text
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = en.AnalyzerName

	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	entryMapping := bleve.NewDocumentMapping()
	entryMapping.AddFieldMappingsAt("tag", englishTextFieldMapping)
	entryMapping.AddFieldMappingsAt("group", englishTextFieldMapping)
	entryMapping.AddFieldMappingsAt("env", englishTextFieldMapping)
	entryMapping.AddFieldMappingsAt("host", keywordFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("indexDocument", entryMapping)

	indexMapping.DefaultAnalyzer = "en"

	return indexMapping
}

func createIndex(suffix string, resultConfig *Config) (bleve.Index, error) {
	defer timeTrack(time.Now(), "createIndex")
	indexName := "jimdex_" + suffix
	indexDir := filepath.Join(files.GetJimConfigDir(), "indices")
	indexPath := filepath.Join(indexDir, indexName)

	index, err := bleve.Open(indexPath)
	if err == nil {
		log.Printf("Reusing existing index %s", indexPath)
		return index, nil
	}
	log.Printf("Error opening the index: %s", err)

	log.Printf("Creating a new index at %s", indexPath)
	// create a new index
	indexMapping := buildIndexMapping()
	index, err = bleve.New(indexPath, indexMapping)
	if err != nil {
		log.Printf("Failed to create a new index %s: %s", indexPath, err)
		return nil, err
	}

	err = indexDocuments(index, resultConfig)
	if err != nil {
		go cleanUpUnusedIndices("", indexDir)
		return nil, err
	}

	go cleanUpUnusedIndices(indexName, indexDir)
	return index, nil
}

func indexDocuments(index bleve.Index, resultConfig *Config) error {
	defer timeTrack(time.Now(), "indexDocuments")

	batch := index.NewBatch()
	batchCount := 0
	for i, entry := range *resultConfig {
		err := batch.Index(strconv.Itoa(i), &indexDocument{
			Group: entry.Group,
			Env:   entry.Env,
			Tag:   entry.Tag,
			Host:  entry.Server.Host,
		})
		if err != nil {
			return err
		}
		batchCount++

		if batchCount > 100 {
			err := index.Batch(batch)
			if err != nil {
				return err
			}
			batch = index.NewBatch()
			batchCount = 0
		}
	}
	// also index last batch
	if batch.Size() > 0 {
		err := index.Batch(batch)
		if err != nil {
			return err
		}
	}
	return nil
}

func cleanUpUnusedIndices(exceptForIndex string, indexDirectory string) {
	dir, err := ioutil.ReadDir(indexDirectory)
	if err != nil {
		log.Printf("Failed to clean up old indices: %s", err)
	}
	for _, d := range dir {
		if exceptForIndex == d.Name() {
			// don't delete the current index
			continue
		}
		// remove all others
		log.Printf("Deleting old index %s", d.Name())
		err := os.RemoveAll(path.Join(indexDirectory, d.Name()))
		if err != nil {
			log.Printf("Failed to clean up old index %s: %s", d.Name(), err)
		}
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
