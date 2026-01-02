package config

import "os"

// Version info - set at build time via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// Config holds application configuration
type Config struct {
	Port         string
	DatabasePath string
	AppName      string
	Version      string
	GitCommit    string
	BuildTime    string
}

// Load reads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "3300"),
		DatabasePath: getEnv("DB_PATH", "./data"),
		AppName:      "KV Service",
		Version:      Version,
		GitCommit:    GitCommit,
		BuildTime:    BuildTime,
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
