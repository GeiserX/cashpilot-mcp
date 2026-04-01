package config

import (
	"testing"
)

func TestLoadCashPilotConfig_Defaults(t *testing.T) {
	t.Setenv("CASHPILOT_URL", "")
	t.Setenv("CASHPILOT_API_KEY", "")
	cfg := LoadCashPilotConfig()
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL default: got %q, want %q", cfg.BaseURL, "http://localhost:8080")
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey default: got %q, want empty", cfg.APIKey)
	}
}

func TestLoadCashPilotConfig_EnvOverride(t *testing.T) {
	t.Setenv("CASHPILOT_URL", "http://dashboard:9090")
	t.Setenv("CASHPILOT_API_KEY", "test-key-abc")
	cfg := LoadCashPilotConfig()
	if cfg.BaseURL != "http://dashboard:9090" {
		t.Errorf("BaseURL: got %q, want %q", cfg.BaseURL, "http://dashboard:9090")
	}
	if cfg.APIKey != "test-key-abc" {
		t.Errorf("APIKey: got %q, want %q", cfg.APIKey, "test-key-abc")
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		fallback string
		want     string
	}{
		{
			name:     "returns env value when set",
			key:      "TEST_CASHPILOT_VAR",
			envVal:   "custom",
			fallback: "default",
			want:     "custom",
		},
		{
			name:     "returns default when env empty",
			key:      "TEST_CASHPILOT_EMPTY",
			envVal:   "",
			fallback: "fallback",
			want:     "fallback",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.envVal)
			got := getEnv(tc.key, tc.fallback)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
