package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DBDSN         string
	JWTSecret     string
	AllowedOrigins string
	DevMode       bool
}

func Load() *Config {
	godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		DBDSN:          getEnv("DB_DSN", "davechat:davechat_pass@tcp(localhost:3306)/davechat?parseTime=true&charset=utf8mb4&loc=UTC"),
		JWTSecret:      jwtSecret,
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		DevMode:        os.Getenv("DEV_MODE") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
