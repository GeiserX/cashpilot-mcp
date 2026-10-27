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
	// ReadOnly hides every tool that changes the fleet (CASHPILOT_READ_ONLY).
	ReadOnly bool
}

func LoadCashPilotConfig() CashPilotConfig {
	return CashPilotConfig{
		// The MCP HTTP listener uses :8081, so it never collides with CashPilot's :8080.
		BaseURL:  getEnv("CASHPILOT_URL", "http://localhost:8080"),
		APIKey:   getEnv("CASHPILOT_API_KEY", ""),
		ReadOnly: isTrue(os.Getenv("CASHPILOT_READ_ONLY")),
	}
}

func isTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
