package web

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
)

func getConfigDirectory() string {
	homeDir, _ := homedir.Dir()
	return path.Join(homeDir, audistoHomeDirecotyName)
}

func getConfigFilePath() string {
	return path.Join(getConfigDirectory(), audistoCredentialsFileName)
}

func createConfigFile(data []byte) error {
	confDir := getConfigDirectory()
	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		os.Mkdir(confDir, 0755)
	}

	return ioutil.WriteFile(getConfigFilePath(), data, 0755)
}

func getPersistedCredentials() (username string, password string) {
	var creds Credentials

	data, err := ioutil.ReadFile(getConfigFilePath())
	if err == nil && len(data) > 0 {
		json.Unmarshal(data, &creds)
	}

	return creds.Username, creds.Password
}
