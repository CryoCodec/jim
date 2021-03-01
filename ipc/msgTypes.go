package ipc

const (
	ReqAttemptDecryption = iota + 1
	ResDecryptionFailed
	ReqLoadFile
	ReqCloseConnection
	ReqStatus
	ResReadyToServe
	ResNeedDecryption
	ResSuccess
	ResError
)
