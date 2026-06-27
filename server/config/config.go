package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

var ServerConfig Config

type Config struct {
	ApiBaseUrl       string
	OauthBaseUrl     string
	OauthClientId    string
	OauthRedirectUrl string
	OauthSecret      string
	Port             int
	ConnectionString string
	AdminKey         string
	WorkerInterval   int
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
