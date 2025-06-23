package main

import (
	"os"
	"strconv"
	"time"
)

var (
	APIUrl         = ""
	ListenInterval = ""
)

type Config struct {
	APIUrl   string
	SPDC     string
	INTERVAL time.Duration
}

func LoadConfig() *Config {
	return &Config{
		APIUrl:   getEnv("SPOTIFY_API_URL", APIUrl),
		INTERVAL: time.Duration(getEnvInt("LISTEN_INTERVAL", ListenInterval)) * time.Millisecond,
	}
}

func getEnv(envKey, defaultValue string) string {
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}
	return defaultValue
}

func getEnvInt(envKey, defaultValue string) int {
	if envValue := os.Getenv(envKey); envValue != "" {
		if intValue, err := strconv.Atoi(envValue); err == nil {
			return intValue
		}
	}

	if defaultValue != "" {
		if intValue, err := strconv.Atoi(defaultValue); err == nil {
			return intValue
		}
	}

	return 200
}
