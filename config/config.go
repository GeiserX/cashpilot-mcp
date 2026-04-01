package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func init() {
	if strings.ToLower(os.Getenv("TRANSPORT")) != "stdio" {
		_ = godotenv.Load()
	}
}

type CashPilotConfig struct {
	BaseURL string
	APIKey  string
}

func LoadCashPilotConfig() CashPilotConfig {
	return CashPilotConfig{
		BaseURL: getEnv("CASHPILOT_URL", "http://localhost:8080"),  // NOTE: MCP HTTP listens on :8081 to avoid conflict
		APIKey:  getEnv("CASHPILOT_API_KEY", ""),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
