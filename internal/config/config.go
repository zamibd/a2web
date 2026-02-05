package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	JWTSecret     []byte
	DBPath        string
	AllowedOrigin string
}

func LoadConfig() *Config {
	// Load .env file if it exists, ignore error (mostly for local dev)
	_ = godotenv.Load()

	return &Config{
		Port:          getEnv("PORT", "8080"),
		JWTSecret:     []byte(getEnv("JWT_SECRET", "secret_key_change_this_later")), // Default for dev, override in prod
		DBPath:        getEnv("DB_PATH", "./storage/audio_streamer.db"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"), // Comma separated for multiple, or * for all
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// Helper to check origins
func (c *Config) IsOriginAllowed(origin string) bool {
	if c.AllowedOrigin == "*" {
		return true
	}
	allowed := strings.Split(c.AllowedOrigin, ",")
	for _, a := range allowed {
		if strings.EqualFold(strings.TrimSpace(a), origin) {
			return true
		}
	}
	return false
}

var AppConfig = LoadConfig()
