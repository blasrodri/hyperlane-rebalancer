package generator

import (
	"testing"

	"github.com/celestiaorg/celestia-rebalancer/pkg/types"
)

func TestParseTokenID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{
			name:    "valid hex with 0x prefix",
			input:   "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			wantErr: false,
			wantLen: 32,
		},
		{
			name:    "valid hex without 0x prefix",
			input:   "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			wantErr: false,
			wantLen: 32,
		},
		{
			name:    "short hex (invalid - must be 32 bytes)",
			input:   "0x1234",
			wantErr: true,
		},
		{
			name:    "invalid hex characters",
			input:   "0xZZZZ",
			wantErr: true,
		},
		{
			name:    "empty string (invalid - must be 32 bytes)",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTokenID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTokenID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("parseTokenID() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestParseAndPadAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantLen int
	}{
		{
			name:    "EVM address with 0x",
			input:   "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			wantErr: false,
			wantLen: 32,
		},
		{
			name:    "EVM address without 0x",
			input:   "742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
			wantErr: true, // Will fail because it's not hex without 0x and not valid bech32
		},
		{
			name:    "short EVM address (gets padded)",
			input:   "0x1234",
			wantErr: false,
			wantLen: 32,
		},
		{
			name:    "cosmos bech32 address",
			input:   "celestia1abc123def456ghi789",
			wantErr: true, // Will fail with invalid bech32
		},
		{
			name:    "empty address",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid hex in EVM address",
			input:   "0xZZZZ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAndPadAddress(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAndPadAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("parseAndPadAddress() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	gen := NewGenerator("celestia1multisig123...")

	routes := &types.Routes{
		Routes: []types.HyperlaneRoute{
			{
				TxHash:      "ABC123",
				BlockHeight: 1000000,
				From:        "celestia1sender",
				Amount:      "1000000",
				Denom:       "utia",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 1380012617,
					Recipient:         "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
					TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				},
			},
		},
		TotalAmount:  "1000000",
		MultisigAddr: "celestia1multisig123...",
	}

	msgs, err := gen.Generate(routes)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("Generate() returned %d messages, want 1", len(msgs))
	}
}

func TestGenerateWithInvalidAmount(t *testing.T) {
	gen := NewGenerator("celestia1multisig123...")

	routes := &types.Routes{
		Routes: []types.HyperlaneRoute{
			{
				TxHash:      "ABC123",
				BlockHeight: 1000000,
				From:        "celestia1sender",
				Amount:      "invalid_amount",
				Denom:       "utia",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 1380012617,
					Recipient:         "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
					TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				},
			},
		},
		TotalAmount:  "1000000",
		MultisigAddr: "celestia1multisig123...",
	}

	_, err := gen.Generate(routes)
	if err == nil {
		t.Error("Generate() expected error for invalid amount, got nil")
	}
}

func TestGenerateWithInvalidTokenID(t *testing.T) {
	gen := NewGenerator("celestia1multisig123...")

	routes := &types.Routes{
		Routes: []types.HyperlaneRoute{
			{
				TxHash:      "ABC123",
				BlockHeight: 1000000,
				From:        "celestia1sender",
				Amount:      "1000000",
				Denom:       "utia",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 1380012617,
					Recipient:         "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
					TokenID:           "invalid_token_id",
				},
			},
		},
		TotalAmount:  "1000000",
		MultisigAddr: "celestia1multisig123...",
	}

	_, err := gen.Generate(routes)
	if err == nil {
		t.Error("Generate() expected error for invalid token ID, got nil")
	}
}

func TestGenerateWithMultipleRoutes(t *testing.T) {
	gen := NewGenerator("celestia1multisig123...")

	routes := &types.Routes{
		Routes: []types.HyperlaneRoute{
			{
				TxHash:      "ABC123",
				BlockHeight: 1000000,
				From:        "celestia1sender1",
				Amount:      "1000000",
				Denom:       "utia",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 1380012617,
					Recipient:         "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
					TokenID:           "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				},
			},
			{
				TxHash:      "DEF456",
				BlockHeight: 1000001,
				From:        "celestia1sender2",
				Amount:      "2000000",
				Denom:       "utia",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 137,
					Recipient:         "0x8b3192f5eebad91f18723bc860fdb53c27af10ab",
					TokenID:           "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				},
			},
		},
		TotalAmount:  "3000000",
		MultisigAddr: "celestia1multisig123...",
	}

	msgs, err := gen.Generate(routes)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(msgs) != 2 {
		t.Errorf("Generate() returned %d messages, want 2", len(msgs))
	}
}
