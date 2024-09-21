package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	StunPort int    `json:"stun_port"`
	DBType   string `json:"db_type"`
	DBConn   string `json:"db_conn"`
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
