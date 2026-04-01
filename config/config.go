package config

import (
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env in the working directory; ignore error if the file is absent.
	_ = godotenv.Load()
}

type CashPilotConfig struct {
	BaseURL string
	APIKey  string
}

func LoadCashPilotConfig() CashPilotConfig {
	return CashPilotConfig{
		BaseURL: getEnv("CASHPILOT_URL", "http://localhost:8080"),
		APIKey:  getEnv("CASHPILOT_API_KEY", ""),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
