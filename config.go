package main

import (
	"os"
	"strconv"
)

var (
	SP_DC             = ""
	RefetchInterval   = ""
	MinListenInterval = ""

	DefaultRefetchInterval   = 600 // in seconds
	DefaultMinListenInterval = 50  // in miliseconds
)

type Config struct {
	SP_DC               string // API for lyrics
	REFETCH_INTERVAL    int    // "404" cache expiration time, in seconds
	MIN_LISTEN_INTERVAL int    // minimum interval between two lyrics fetches, in milliseconds
}

func LoadConfig() *Config {
	return &Config{
		SP_DC:               getEnv("SPOTIFY_API_URL", SP_DC, ""),
		REFETCH_INTERVAL:    getEnvInt("REFETCH_INTERVAL", RefetchInterval, DefaultRefetchInterval),
		MIN_LISTEN_INTERVAL: getEnvInt("MIN_LISTEN_INTERVAL", MinListenInterval, DefaultMinListenInterval),
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
