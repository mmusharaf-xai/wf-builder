package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port     string
	Database DatabaseConfig
	CORS     CORSConfig
	// Hardcoded user for parity with Next.js app until auth is split out.
	DefaultUserID string
}

type DatabaseConfig struct {
	URL string
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() (*Config, error) {
	dbURL := strings.TrimSpace(os.Getenv("PG_DB_URL"))
	if dbURL == "" {
		return nil, fmt.Errorf("PG_DB_URL is required")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	origins := []string{"http://localhost:3000", "http://localhost:3001"}
	if raw := strings.TrimSpace(os.Getenv("CORS_ORIGINS")); raw != "" {
		parts := strings.Split(raw, ",")
		origins = origins[:0]
		for _, p := range parts {
			if o := strings.TrimSpace(p); o != "" {
				origins = append(origins, o)
			}
		}
	}

	userID := strings.TrimSpace(os.Getenv("DEFAULT_USER_ID"))
	if userID == "" {
		userID = "1"
	}

	return &Config{
		Port: port,
		Database: DatabaseConfig{
			URL: dbURL,
		},
		CORS: CORSConfig{
			AllowedOrigins: origins,
		},
		DefaultUserID: userID,
	}, nil
}

func EnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
