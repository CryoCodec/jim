package config

import (
	"github.com/CryoCodec/jim/files"
	"path/filepath"
)

const (
	Protocol = "unix"
)

// GetSocketAddress returns the address of jim's UDS socket
func GetSocketAddress() string {
	return filepath.Join(files.GetJimConfigDir(), "socket")
}
