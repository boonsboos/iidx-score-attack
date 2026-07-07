package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

var ServerConfig Config

type Config struct {
	Apis             []ApiConfig
	Port             int
	ConnectionString string
	AdminKey         string
	WorkerInterval   int
	EncryptionKey    string
}

type ApiConfig struct {
	ApiBaseUrl       string
	AuthBaseUrl      string
	ClientId         string
	Secret           string
	Name             string
	OauthRedirectUrl string
}

func InitConfig() {

	log.Println("Loading config...")

	jsonFile, err := os.Open("config.local.json")
	if err != nil {
		log.Println("Couldn't open config file:", err)
	}

	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Println("Couldn't read config file:", err)
		return
	}

	err = json.Unmarshal(byteValue, &ServerConfig)
	if err != nil {
		log.Println("Couldn't deserialize config:", err)
		return
	}

	log.Println("Loading config OK")
}

func (c Config) GetApiConfigByName(name string) *ApiConfig {
	for _, apiConfig := range c.Apis {
		if apiConfig.Name == name {
			return &apiConfig
		}
	}
	return nil
}
