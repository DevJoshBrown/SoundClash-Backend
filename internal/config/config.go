package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	ClerkSecretKey string
	AllowedOrigin  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
		AllowedOrigin:  os.Getenv("ALLOWED_ORIGIN"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
