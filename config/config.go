package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type ConfStructure struct {
	TTLMinutes              int `yaml:"ttlMinutes"`
	CheckIntervalSeconds    int `yaml:"checkIntervalSeconds"`
	WsUpdateIntervalSeconds int `yaml:"wsUpdateIntervalSeconds"`
}

var Config = &ConfStructure{}

func ReadConfig(configPath string) {

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		println("Failed to read the config file: ", err)
		return
	}

	// Parse the YAML content into Config struct
	err = yaml.Unmarshal(data, Config)
	if err != nil {
		println("Failed to parse the config file: ", err)
		return
	}
}
