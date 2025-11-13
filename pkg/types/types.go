package types

import (
	"encoding/json"
	"fmt"
)

// HyperlaneRoute represents routing information extracted from MsgRemoteTransfer custom_hook_metadata
type HyperlaneRoute struct {
	// Source transaction information
	TxHash      string `json:"tx_hash"`
	BlockHeight int64  `json:"block_height"`
	From        string `json:"from"`
	Amount      string `json:"amount"`
	Denom       string `json:"denom"`
	CustomHookMetadata string `json:"custom_hook_metadata"`

	// Parsed Hyperlane routing info
	RouteInfo *RouteInfo `json:"route_info,omitempty"`
}

// RouteInfo contains the parsed Hyperlane routing information from custom_hook_metadata
type RouteInfo struct {
	DestinationDomain uint32 `json:"destination_domain"`
	Recipient         string `json:"recipient"`
	TokenID           string `json:"token_id"`
	Amount            string `json:"amount,omitempty"` // Optional: overrides the received amount
}

// ParseCustomHookMetadata attempts to parse the custom_hook_metadata as JSON containing RouteInfo
func ParseCustomHookMetadata(metadata string) (*RouteInfo, error) {
	var routeInfo RouteInfo
	if err := json.Unmarshal([]byte(metadata), &routeInfo); err != nil {
		return nil, fmt.Errorf("failed to parse custom_hook_metadata as JSON: %w", err)
	}

	// Validate required fields
	if routeInfo.DestinationDomain == 0 {
		return nil, fmt.Errorf("destination_domain is required")
	}
	if routeInfo.Recipient == "" {
		return nil, fmt.Errorf("recipient is required")
	}
	if routeInfo.TokenID == "" {
		return nil, fmt.Errorf("token_id is required")
	}

	return &routeInfo, nil
}

// Routes is a collection of HyperlaneRoute with metadata
type Routes struct {
	Routes       []HyperlaneRoute `json:"routes"`
	TotalAmount  string           `json:"total_amount"`
	MultisigAddr string           `json:"multisig_address"`
}

// DomainConfig maps chain names to Hyperlane domain IDs
type DomainConfig struct {
	Domains map[string]uint32 `json:"domains"`
}

// Common Hyperlane domain IDs (can be extended via config)
var DefaultDomains = map[string]uint32{
	"ethereum":  1,
	"polygon":   137,
	"avalanche": 43114,
	"bsc":       56,
	"arbitrum":  42161,
	"optimism":  10,
	"moonbeam":  1284,
	"gnosis":    100,
	"celo":      42220,
	"scroll":    534352,
	"celestia":  69420, // From test
}
