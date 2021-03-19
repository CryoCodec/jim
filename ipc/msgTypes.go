package ipc

// Denotes the message types uesd for server communication
const (
	ReqAttemptDecryption = iota + 1
	ResDecryptionFailed
	ResJsonDeserializationFailed
	ReqLoadFile
	ResRequireConfigFile
	ReqListEntries
	ResListEntries
	ReqClosestMatch
	ResClosestMatch
	ResNoMatch
	ReqClosestN
	ResClosestN
	ReqCloseConnection
	ReqStatus
	ResReadyToServe
	ResNeedDecryption
	ResSuccess
	ResError
)

// Code denotes the message's code, a simple int
type Code int

// Message is the type that resembles a domain specific message
type Message struct {
	Code    Code
	Payload []byte
}

var msgCodeToString = map[uint16]string{
	ReqAttemptDecryption:         `REQ_ATTEMPT_DECRYPTION`,
	ResDecryptionFailed:          `RES_DECRYPTION_FAILED`,
	ResJsonDeserializationFailed: `RES_JSON_DESERIALIZATION_FAILED`,
	ReqLoadFile:                  `REQ_LOAD_FILE`,
	ResRequireConfigFile:         `RES_REQUIRE_CONFIG_FILE`,
	ReqCloseConnection:           `REQ_CLOSE_CONNECTION`,
	ReqStatus:                    `REQ_STATUS`,
	ResReadyToServe:              `RES_READY_TO_SERVE`,
	ResNeedDecryption:            `RES_NEED_DECRYPTION`,
	ResSuccess:                   `RES_SUCCESS`,
	ResError:                     `RES_ERROR`,
	ResListEntries:               `RES_LIST_ENTRIES`,
	ReqListEntries:               `REQ_LIST_ENTRIES`,
	ReqClosestMatch:              `REQ_CLOSEST_MATCH`,
	ResClosestMatch:              `RES_CLOSEST_MATCH`,
	ResNoMatch:                   `RES_NO_MATCH`,
}
