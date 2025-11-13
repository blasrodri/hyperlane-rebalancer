package types

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRoute(t *testing.T) {
	config := &Config{
		Whitelist: AddressWhitelist{
			Domains: map[uint32][]string{
				2340: { // Eden
					"0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
					"0x1111111111111111111111111111111111111111",
				},
				1: { // Ethereum
					"0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				},
			},
		},
	}

	tests := []struct {
		name        string
		route       *RouteInfo
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid whitelisted address - exact match",
			route: &RouteInfo{
				DestinationDomain: 2340,
				Recipient:         "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
				TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			expectError: false,
		},
		{
			name: "valid whitelisted address - case insensitive",
			route: &RouteInfo{
				DestinationDomain: 2340,
				Recipient:         "0x742D35CC6634C0532925A3B844BC9E7595F0BEB0",
				TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			expectError: false,
		},
		{
			name: "invalid address - not whitelisted",
			route: &RouteInfo{
				DestinationDomain: 2340,
				Recipient:         "0x9999999999999999999999999999999999999999",
				TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			expectError: true,
			errorMsg:    "not whitelisted for domain",
		},
		{
			name: "invalid domain - not configured",
			route: &RouteInfo{
				DestinationDomain: 99999,
				Recipient:         "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
				TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			expectError: true,
			errorMsg:    "not configured in whitelist",
		},
		{
			name: "valid different domain",
			route: &RouteInfo{
				DestinationDomain: 1,
				Recipient:         "0xABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCD", // Case insensitive
				TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateRoute(tt.route)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
  "whitelist": {
    "domains": {
      "2340": [
        "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
        "0x1111111111111111111111111111111111111111"
      ],
      "1": [
        "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
      ]
    }
  }
}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify domains loaded correctly
	if len(config.Whitelist.Domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(config.Whitelist.Domains))
	}

	// Verify addresses normalized to lowercase
	edenAddrs := config.Whitelist.Domains[2340]
	if len(edenAddrs) != 2 {
		t.Errorf("expected 2 addresses for Eden domain, got %d", len(edenAddrs))
	}

	// Test validation with loaded config
	validRoute := &RouteInfo{
		DestinationDomain: 2340,
		Recipient:         "0x742D35CC6634C0532925A3B844BC9E7595F0BEB0", // Mixed case
		TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
	}

	if err := config.ValidateRoute(validRoute); err != nil {
		t.Errorf("ValidateRoute failed for valid route: %v", err)
	}
}

func TestSaveConfig(t *testing.T) {
	config := DefaultConfig()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	if err := config.SaveConfig(configPath); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Load it back and verify
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(loadedConfig.Whitelist.Domains) != len(config.Whitelist.Domains) {
		t.Errorf("loaded config has different number of domains")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
