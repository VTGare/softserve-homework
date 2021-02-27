package config

import (
	"encoding/json"
	"os"
)

//Config contains all application configuration.
type Config struct {
	Host  string `json:"host"`
	Port  string `json:"port"`
	Redis struct {
		Host string `json:"host"`
		Port string `json:"port"`
	} `json:"redis"`
}

//New returns a new Config from a file located in path.
func New(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
