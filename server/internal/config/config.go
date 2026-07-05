package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DBDSN          string
	JWTSecret      string
	AllowedOrigins string
	DevMode        bool
	MaxFileSize    int64
	TURNURL        string
	TURNUsername   string
	TURNPassword   string
}

func Load() *Config {
	godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		DBDSN:          getEnv("DATABASE_DSN", "davechat:davechat_pass@tcp(192.168.101.133:3307)/davechat?parseTime=true&charset=utf8mb4&loc=UTC"),
		JWTSecret:      jwtSecret,
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		DevMode:        os.Getenv("DEV_MODE") == "true",
		MaxFileSize: getEnvInt("MAX_FILE_SIZE", 5242880),
		TURNURL:        getEnv("TURN_URL", ""),
		TURNUsername:   getEnv("TURN_USERNAME", ""),
		TURNPassword:   getEnv("TURN_PASSWORD", ""),
	}
}

func getEnvInt(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
