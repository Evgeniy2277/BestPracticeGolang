package config

import (
	"os"
	"strconv"
)

type Config struct {
	MaxDepth   int
	MaxResults int
	MaxErrors  int
	Url        string
	Timeout    int
	Time       int
}

// New returns a new Config struct
func New() *Config {
	return &Config{

		MaxDepth:   getEnvAsInt("MAX_DEPTH", 1),
		MaxResults: getEnvAsInt("MAX_RESULTS", 1),
		MaxErrors:  getEnvAsInt("MAX_ERRORS", 1),
		Url:        getEnvAsString("URL", " "),
		Timeout:    getEnvAsInt("TIMEOUT", 1),
		Time:       getEnvAsInt("TIME", 1),
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

// Simple helper function to read an environment variable into integer or return a default value
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

func getEnvAsString(name string, defaultVal string) string {
	valStr := getEnv(name, "")

	if valStr == "" {
		return defaultVal
	}

	return valStr
}
