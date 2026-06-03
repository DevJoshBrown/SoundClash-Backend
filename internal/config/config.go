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
	R2AccessKey    string
	R2Bucket       string
	R2Endpoint     string
	R2SecretKey    string
	R2PublicURL    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
		AllowedOrigin:  os.Getenv("ALLOWED_ORIGIN"),
		R2AccessKey:    os.Getenv("R2_ACCESS_KEY_ID"),
		R2Bucket:       os.Getenv("R2_Bucket"),
		R2Endpoint:     os.Getenv("R2_ENDPOINT"),
		R2SecretKey:    os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2PublicURL:    os.Getenv("R2_PUBLIC_URL"),
	}
	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
