package main

import (
	"os"
	"strconv"
	"time"
)

var (
	APIUrl          = ""
	ListenInterval  = ""
	RefetchInterval = ""

	DefaultListenInterval  = 200 // in ms
	DefaultRefetchInterval = 600 // in seconds
)

type Config struct {
	APIUrl           string        // API for lyrics
	LISTEN_INTERVAL  time.Duration // sleep interval for listen mode
	REFETCH_INTERVAL int           // "404" cache expiration time, in seconds
}

func LoadConfig() *Config {
	return &Config{
		APIUrl:           getEnv("SPOTIFY_API_URL", APIUrl, ""),
		LISTEN_INTERVAL:  time.Duration(getEnvInt("LISTEN_INTERVAL", ListenInterval, DefaultListenInterval)) * time.Millisecond,
		REFETCH_INTERVAL: getEnvInt("REFETCH_INTERVAL", RefetchInterval, DefaultRefetchInterval),
	}
}

func getEnv(envKey string, compileValue string, defaultValue string) string {
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}
	if compileValue != "" {
		return compileValue
	}
	return defaultValue
}

func getEnvInt(envKey string, compileValue string, defaultValue int) int {
	if envValue := os.Getenv(envKey); envValue != "" {
		if intValue, err := strconv.Atoi(envValue); err == nil {
			return intValue
		}
	}

	if compileValue != "" {
		if intValue, err := strconv.Atoi(compileValue); err == nil {
			return intValue
		}
	}

	return defaultValue
}
