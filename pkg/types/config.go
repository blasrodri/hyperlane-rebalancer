package types

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// AddressWhitelist defines allowed recipient addresses for each Hyperlane domain
type AddressWhitelist struct {
	// Map of domain ID to list of whitelisted addresses
	Domains map[uint32][]string `json:"domains"`
}

// Config holds the configuration for the rebalancer including address whitelists
type Config struct {
	Whitelist AddressWhitelist `json:"whitelist"`
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Normalize all addresses in whitelist to lowercase for case-insensitive comparison
	for domain, addresses := range config.Whitelist.Domains {
		normalized := make([]string, len(addresses))
		for i, addr := range addresses {
			normalized[i] = normalizeAddress(addr)
		}
		config.Whitelist.Domains[domain] = normalized
	}

	return &config, nil
}

// ValidateRoute validates that the route's recipient address is whitelisted for the destination domain
func (c *Config) ValidateRoute(route *RouteInfo) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	// Get whitelisted addresses for this domain
	whitelistedAddresses, exists := c.Whitelist.Domains[route.DestinationDomain]
	if !exists {
		return fmt.Errorf("domain %d is not configured in whitelist", route.DestinationDomain)
	}

	if len(whitelistedAddresses) == 0 {
		return fmt.Errorf("domain %d has no whitelisted addresses", route.DestinationDomain)
	}

	// Normalize the recipient address for comparison
	normalizedRecipient := normalizeAddress(route.Recipient)

	// Check if recipient is in whitelist
	for _, whitelistedAddr := range whitelistedAddresses {
		if normalizedRecipient == whitelistedAddr {
			return nil // Valid!
		}
	}

	return fmt.Errorf("recipient %s is not whitelisted for domain %d", route.Recipient, route.DestinationDomain)
}

// normalizeAddress normalizes an address for comparison (lowercase, trim 0x prefix)
func normalizeAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	addr = strings.ToLower(addr)
	// Keep 0x prefix if present for consistency
	return addr
}

// DefaultConfig returns a default configuration with example whitelisted addresses
// This should be replaced with actual production addresses
func DefaultConfig() *Config {
	return &Config{
		Whitelist: AddressWhitelist{
			Domains: map[uint32][]string{
				// Eden domain
				2340: {
					"0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
				},
				// Ethereum
				1: {
					"0x1234567890123456789012345678901234567890",
				},
				// Polygon
				137: {
					"0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				},
			},
		},
	}
}

// SaveConfig saves the configuration to a JSON file
func (c *Config) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
