// Package files offers some utility methods for checking file existence or finding jim's config data.
package files

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

// Exists checks whether a file with given filepath exists.
// Returns true, if the file indeed exists, in all other cases false.
func Exists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// GetJimConfigFilePath returns the filepath to the encrypted jim config.
// Highest priority has the env variable JIM_CONFIG_FILE.
// If the variable is not set the default location  ~/.jim/config.json.enc is used.
func GetJimConfigFilePath() (string, error) {
	path := os.Getenv("JIM_CONFIG_FILE") // env variable has highest priority
	if path == "" {                      // fallback to the standard location
		path = filepath.Join(GetJimConfigDir(), "config.json.enc")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf(
			`No encrypted config file was found at the configured path '%s'. 
Either proceed with 
jim doctor
jim encrypt

or set the path to the config file via the environment variable JIM_CONFIG_FILE`, path)
	}
	return path, nil
}

// GetJimConfigDir returns the filepath to jim's config directory ~/.jim
func GetJimConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not read user home directory path")
	}
	return filepath.Join(homeDir, ".jim")
}
