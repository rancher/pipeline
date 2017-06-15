package config

import (
	"os"
)

type Config struct {
	CattleUrl       string
	CattleAccessKey string
	CattleSecretKey string
}

func GetConfig() Config {
	config := Config{
		CattleUrl:       os.Getenv("CATTLE_URL"),
		CattleAccessKey: os.Getenv("CATTLE_ACCESS_KEY"),
		CattleSecretKey: os.Getenv("CATTLE_SECRET_KEY"),
	}

	return config
}
