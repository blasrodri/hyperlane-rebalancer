package types

import (
	"encoding/json"
	"testing"
)

func TestParseCustomHookMetadata(t *testing.T) {
	tests := []struct {
		name        string
		metadata    string
		wantErr     bool
		checkFields func(*testing.T, *RouteInfo)
	}{
		{
			name: "valid metadata with all fields",
			metadata: `{"destination_domain": 1380012617, "recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0", "token_id": "0x1234567890abcdef", "amount": "1000000"}`,
			wantErr: false,
			checkFields: func(t *testing.T, ri *RouteInfo) {
				if ri.DestinationDomain != 1380012617 {
					t.Errorf("DestinationDomain = %d, want 1380012617", ri.DestinationDomain)
				}
				if ri.Recipient != "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0" {
					t.Errorf("Recipient = %s, want 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0", ri.Recipient)
				}
				if ri.TokenID != "0x1234567890abcdef" {
					t.Errorf("TokenID = %s, want 0x1234567890abcdef", ri.TokenID)
				}
				if ri.Amount != "1000000" {
					t.Errorf("Amount = %s, want 1000000", ri.Amount)
				}
			},
		},
		{
			name: "valid metadata without amount field",
			metadata: `{"destination_domain": 137, "recipient": "0x8b3192f5eebad91f18723bc860fdb53c27af10ab", "token_id": "0xabcdef"}`,
			wantErr: false,
			checkFields: func(t *testing.T, ri *RouteInfo) {
				if ri.DestinationDomain != 137 {
					t.Errorf("DestinationDomain = %d, want 137", ri.DestinationDomain)
				}
				if ri.Amount != "" {
					t.Errorf("Amount = %s, want empty string", ri.Amount)
				}
			},
		},
		{
			name:    "invalid JSON",
			metadata:    `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "missing destination_domain",
			metadata:    `{"recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0", "token_id": "0x1234"}`,
			wantErr: true,
		},
		{
			name:    "missing recipient",
			metadata:    `{"destination_domain": 1380012617, "token_id": "0x1234"}`,
			wantErr: true,
		},
		{
			name:    "missing token_id",
			metadata:    `{"destination_domain": 1380012617, "recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"}`,
			wantErr: true,
		},
		{
			name:    "empty metadata",
			metadata:    ``,
			wantErr: true,
		},
		{
			name: "cosmos bech32 recipient",
			metadata: `{"destination_domain": 69420, "recipient": "celestia1abc123def456", "token_id": "0xabcd"}`,
			wantErr: false,
			checkFields: func(t *testing.T, ri *RouteInfo) {
				if ri.Recipient != "celestia1abc123def456" {
					t.Errorf("Recipient = %s, want celestia1abc123def456", ri.Recipient)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCustomHookMetadata(tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCustomHookMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFields != nil {
				tt.checkFields(t, got)
			}
		})
	}
}

func TestRouteInfoJSON(t *testing.T) {
	original := RouteInfo{
		DestinationDomain: 1380012617,
		Recipient:         "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
		TokenID:           "0x1234567890abcdef",
		Amount:            "1000000",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal back
	var decoded RouteInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Compare
	if decoded.DestinationDomain != original.DestinationDomain {
		t.Errorf("DestinationDomain = %d, want %d", decoded.DestinationDomain, original.DestinationDomain)
	}
	if decoded.Recipient != original.Recipient {
		t.Errorf("Recipient = %s, want %s", decoded.Recipient, original.Recipient)
	}
	if decoded.TokenID != original.TokenID {
		t.Errorf("TokenID = %s, want %s", decoded.TokenID, original.TokenID)
	}
	if decoded.Amount != original.Amount {
		t.Errorf("Amount = %s, want %s", decoded.Amount, original.Amount)
	}
}

func TestHyperlaneRouteJSON(t *testing.T) {
	route := HyperlaneRoute{
		TxHash:             "ABC123",
		BlockHeight:        1000000,
		From:               "celestia1sender",
		Amount:             "1000000",
		Denom:              "utia",
		CustomHookMetadata: `{"destination_domain": 137, "recipient": "0x123", "token_id": "0xabc"}`,
		RouteInfo: &RouteInfo{
			DestinationDomain: 137,
			Recipient:         "0x123",
			TokenID:           "0xabc",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(route)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal back
	var decoded HyperlaneRoute
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Compare
	if decoded.TxHash != route.TxHash {
		t.Errorf("TxHash = %s, want %s", decoded.TxHash, route.TxHash)
	}
	if decoded.RouteInfo == nil {
		t.Fatal("RouteInfo is nil")
	}
	if decoded.RouteInfo.DestinationDomain != route.RouteInfo.DestinationDomain {
		t.Errorf("DestinationDomain = %d, want %d", decoded.RouteInfo.DestinationDomain, route.RouteInfo.DestinationDomain)
	}
}
