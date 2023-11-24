package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Registries []string `json:"Registries"`
}

const confFilePath = "/etc/my-container/config.json"

var defaultConfig = Config{Registries: []string{"docker.io"}}
var GlobalConfig Config

func init() {
	bytes, err := os.ReadFile(confFilePath)
	if err != nil {
		GlobalConfig = defaultConfig
		return
	}
	if err := json.Unmarshal(bytes, &GlobalConfig); err != nil {
		GlobalConfig = defaultConfig
	}
}
