package files

import (
	"log"
	"os"
	"path/filepath"
)

func Exists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func GetJimConfigFilePath() string {
	path := os.Getenv("JIM_CONFIG_FILE") // env variable has highest priority
	if path == "" {                      // fallback to the standard location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Could not read user home directory path")
		}
		path = filepath.Join(homeDir, ".jim", "config.json.enc")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatal(
				`No encrypted config file was found at the expected path ~/.jim/config.json.enc. 
Either proceed with 
jim doctor
jim encrypt

or set the path to the config file via the environment variable JIM_CONFIG_FILE`)
		}
	}
	return path
}

func GetJimConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not read user home directory path")
	}
	return filepath.Join(homeDir, ".jim")
}
