package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

func Load() (models.Config, error) {
	var cfg models.Config
	// Load .env file (optional fallback if not found)
	err := godotenv.Load(".env")
	if err != nil {
		// Log but don’t fail if .env is missing
		// return err
	}

	// Read and parse PORT
	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return cfg, err
		}
		cfg.Port = port
	} else {
		cfg.Port = 8080 // Default port
	}

	cfg.Env = os.Getenv("ENV")

	// JWT settings
	cfg.JWT.SecretKey = os.Getenv("JWT_SECRET_KEY")
	cfg.JWT.Issuer = os.Getenv("JWT_ISSUER")
	cfg.JWT.Audience = os.Getenv("JWT_AUDIENCE")
	cfg.JWT.Algorithm = os.Getenv("JWT_ALGORITHM")

	if expiry := os.Getenv("JWT_EXPIRY"); expiry != "" {
		dur, err := time.ParseDuration(expiry)
		if err != nil {
			return cfg, err
		}
		cfg.JWT.Expiry = dur
	}

	if refresh := os.Getenv("JWT_REFRESH"); refresh != "" {
		dur, err := time.ParseDuration(refresh)
		if err != nil {
			return cfg, err
		}
		cfg.JWT.Refresh = dur
	}

	// DB settings
	cfg.DB.DSN = os.Getenv("DB_DSN")
	cfg.DB.DEVDSN = os.Getenv("DB_DSN_DEV")

	return cfg, nil
}
