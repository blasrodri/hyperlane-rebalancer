package pkg

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	warptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	"github.com/celestiaorg/celestia-rebalancer/pkg/generator"
	"github.com/celestiaorg/celestia-rebalancer/pkg/types"
	"github.com/celestiaorg/celestia-rebalancer/pkg/verifier"
)

// TestEndToEndRebalancing tests the complete flow:
// 1. Parse custom_hook_metadata from incoming MsgRemoteTransfer
// 2. Generate MsgRemoteTransfer
// 3. Verify transaction matches intent
func TestEndToEndRebalancing(t *testing.T) {
	// STEP 1: Simulate incoming MsgRemoteTransfer with routing metadata
	incomingCustomHookMetadata := `{
		"destination_domain": 1380012617,
		"recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
		"token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"amount": "1000000"
	}`

	// STEP 2: Parse the custom_hook_metadata
	routeInfo, err := types.ParseCustomHookMetadata(incomingCustomHookMetadata)
	if err != nil {
		t.Fatalf("Failed to parse custom_hook_metadata: %v", err)
	}

	// Verify parsed data
	if routeInfo.DestinationDomain != 1380012617 {
		t.Errorf("DestinationDomain = %d, want 1380012617", routeInfo.DestinationDomain)
	}
	if routeInfo.Recipient != "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0" {
		t.Errorf("Recipient = %s, want 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0", routeInfo.Recipient)
	}

	// STEP 3: Create a HyperlaneRoute (simulating what parser would extract)
	route := types.HyperlaneRoute{
		TxHash:             "ABC123DEF456",
		BlockHeight:        1000000,
		From:               "celestia1sender123...",
		Amount:             "1000000",
		Denom:              "utia",
		CustomHookMetadata: incomingCustomHookMetadata,
		RouteInfo:          routeInfo,
	}

	routes := &types.Routes{
		Routes:       []types.HyperlaneRoute{route},
		TotalAmount:  "1000000",
		MultisigAddr: "celestia1multisigabc...",
	}

	// STEP 4: Generate MsgRemoteTransfer transaction
	gen := generator.NewGenerator("celestia1multisigabc...")
	msgs, err := gen.Generate(routes)
	if err != nil {
		t.Fatalf("Failed to generate transaction: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	// Cast to MsgRemoteTransfer
	remoteTransferMsg, ok := msgs[0].(*warptypes.MsgRemoteTransfer)
	if !ok {
		t.Fatalf("Message is not MsgRemoteTransfer")
	}

	// STEP 5: Verify the generated transaction
	// Check sender
	if remoteTransferMsg.Sender != "celestia1multisigabc..." {
		t.Errorf("Sender = %s, want celestia1multisigabc...", remoteTransferMsg.Sender)
	}

	// Check destination domain
	if remoteTransferMsg.DestinationDomain != 1380012617 {
		t.Errorf("DestinationDomain = %d, want 1380012617", remoteTransferMsg.DestinationDomain)
	}

	// Check amount
	expectedAmount := math.NewInt(1000000)
	if !remoteTransferMsg.Amount.Equal(expectedAmount) {
		t.Errorf("Amount = %s, want %s", remoteTransferMsg.Amount.String(), expectedAmount.String())
	}

	// Check token ID (32 bytes)
	if len(remoteTransferMsg.TokenId) != 32 {
		t.Errorf("TokenId length = %d, want 32", len(remoteTransferMsg.TokenId))
	}

	// Check recipient (32 bytes, padded)
	if len(remoteTransferMsg.Recipient) != 32 {
		t.Errorf("Recipient length = %d, want 32", len(remoteTransferMsg.Recipient))
	}

	t.Logf("✓ Successfully created MsgRemoteTransfer:")
	t.Logf("  Sender: %s", remoteTransferMsg.Sender)
	t.Logf("  Destination Domain: %d", remoteTransferMsg.DestinationDomain)
	t.Logf("  Amount: %s", remoteTransferMsg.Amount.String())
	t.Logf("  Token ID: %x", remoteTransferMsg.TokenId[:])
	t.Logf("  Recipient: %x", remoteTransferMsg.Recipient[:])
}

// TestMultipleTransfers tests handling multiple incoming transfers
func TestMultipleTransfers(t *testing.T) {
	// Simulate two incoming MsgRemoteTransfer messages
	transfers := []struct {
		txHash             string
		customHookMetadata string
		amount             string
	}{
		{
			txHash: "TX1_ABC123",
			customHookMetadata: `{
				"destination_domain": 1380012617,
				"recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
				"token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			}`,
			amount: "1000000",
		},
		{
			txHash: "TX2_DEF456",
			customHookMetadata: `{
				"destination_domain": 137,
				"recipient": "0x8b3192f5eebad91f18723bc860fdb53c27af10ab",
				"token_id": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
			}`,
			amount: "2000000",
		},
	}

	// Parse and create routes
	var routes []types.HyperlaneRoute
	totalAmount := math.ZeroInt()

	for _, transfer := range transfers {
		routeInfo, err := types.ParseCustomHookMetadata(transfer.customHookMetadata)
		if err != nil {
			t.Fatalf("Failed to parse custom_hook_metadata for %s: %v", transfer.txHash, err)
		}

		route := types.HyperlaneRoute{
			TxHash:             transfer.txHash,
			BlockHeight:        1000000,
			From:               "celestia1sender...",
			Amount:             transfer.amount,
			Denom:              "utia",
			CustomHookMetadata: transfer.customHookMetadata,
			RouteInfo:          routeInfo,
		}

		routes = append(routes, route)

		amt, ok := math.NewIntFromString(transfer.amount)
		if ok {
			totalAmount = totalAmount.Add(amt)
		}
	}

	allRoutes := &types.Routes{
		Routes:       routes,
		TotalAmount:  totalAmount.String(),
		MultisigAddr: "celestia1multisig...",
	}

	// Generate transactions
	gen := generator.NewGenerator("celestia1multisig...")
	msgs, err := gen.Generate(allRoutes)
	if err != nil {
		t.Fatalf("Failed to generate transactions: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(msgs))
	}

	// Verify first message
	msg1, ok := msgs[0].(*warptypes.MsgRemoteTransfer)
	if !ok {
		t.Fatalf("Message 0 is not MsgRemoteTransfer")
	}
	if msg1.DestinationDomain != 1380012617 {
		t.Errorf("Message 0: DestinationDomain = %d, want 1380012617", msg1.DestinationDomain)
	}

	// Verify second message
	msg2, ok := msgs[1].(*warptypes.MsgRemoteTransfer)
	if !ok {
		t.Fatalf("Message 1 is not MsgRemoteTransfer")
	}
	if msg2.DestinationDomain != 137 {
		t.Errorf("Message 1: DestinationDomain = %d, want 137", msg2.DestinationDomain)
	}

	t.Logf("✓ Successfully created %d MsgRemoteTransfer messages", len(msgs))
	t.Logf("  Total amount routed: %s utia", totalAmount.String())
}

// TestCustomHookMetadataToTransactionRoundTrip tests that we can verify generated transactions
func TestCustomHookMetadataToTransactionRoundTrip(t *testing.T) {
	// Original custom_hook_metadata
	customHookMetadata := `{
		"destination_domain": 1380012617,
		"recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
		"token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"amount": "5000000"
	}`

	// Parse custom_hook_metadata
	routeInfo, err := types.ParseCustomHookMetadata(customHookMetadata)
	if err != nil {
		t.Fatalf("Failed to parse custom_hook_metadata: %v", err)
	}

	// Create route
	route := types.HyperlaneRoute{
		TxHash:             "TEST_TX_HASH",
		BlockHeight:        1000000,
		From:               "celestia1sender...",
		Amount:             "5000000",
		Denom:              "utia",
		CustomHookMetadata: customHookMetadata,
		RouteInfo:          routeInfo,
	}

	routes := &types.Routes{
		Routes:       []types.HyperlaneRoute{route},
		TotalAmount:  "5000000",
		MultisigAddr: "celestia1multisig...",
	}

	// Generate transaction
	gen := generator.NewGenerator("celestia1multisig...")
	msgs, err := gen.Generate(routes)
	if err != nil {
		t.Fatalf("Failed to generate transaction: %v", err)
	}

	// Verify using the verifier
	v := verifier.NewVerifier()
	msg, ok := msgs[0].(*warptypes.MsgRemoteTransfer)
	if !ok {
		t.Fatalf("Message is not MsgRemoteTransfer")
	}

	// Manual verification (simulating what the verifier does)
	if !v.MatchesRoute(msg, &route) {
		t.Error("Generated message does not match original route")
	}

	t.Log("✓ Round-trip verification successful!")
	t.Log("  Original custom_hook_metadata → Route → MsgRemoteTransfer → Verified ✓")
}

// TestInvalidCustomHookMetadataHandling tests error handling for invalid custom_hook_metadata
func TestInvalidCustomHookMetadataHandling(t *testing.T) {
	invalidMetadata := []struct {
		name     string
		metadata string
	}{
		{
			name:     "missing destination_domain",
			metadata: `{"recipient": "0x123", "token_id": "0xabc"}`,
		},
		{
			name:     "missing recipient",
			metadata: `{"destination_domain": 123, "token_id": "0xabc"}`,
		},
		{
			name:     "missing token_id",
			metadata: `{"destination_domain": 123, "recipient": "0x123"}`,
		},
		{
			name:     "invalid JSON",
			metadata: `{invalid json}`,
		},
		{
			name:     "empty metadata",
			metadata: ``,
		},
	}

	for _, tc := range invalidMetadata {
		t.Run(tc.name, func(t *testing.T) {
			_, err := types.ParseCustomHookMetadata(tc.metadata)
			if err == nil {
				t.Errorf("Expected error for invalid custom_hook_metadata, got nil")
			} else {
				t.Logf("✓ Correctly rejected invalid custom_hook_metadata: %v", err)
			}
		})
	}
}

// TestJSONSerialization tests that routes can be saved and loaded
func TestJSONSerialization(t *testing.T) {
	// Create a route
	original := types.Routes{
		Routes: []types.HyperlaneRoute{
			{
				TxHash:             "ABC123",
				BlockHeight:        1000000,
				From:               "celestia1sender...",
				Amount:             "1000000",
				Denom:              "utia",
				CustomHookMetadata: `{"destination_domain": 123, "recipient": "0x123", "token_id": "0xabc"}`,
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 123,
					Recipient:         "0x123",
					TokenID:           "0xabc",
				},
			},
		},
		TotalAmount:  "1000000",
		MultisigAddr: "celestia1multisig...",
	}

	// Marshal to JSON (simulating save to file)
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal routes: %v", err)
	}

	t.Logf("Serialized routes:\n%s", string(data))

	// Unmarshal back (simulating load from file)
	var loaded types.Routes
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal routes: %v", err)
	}

	// Verify data integrity
	if len(loaded.Routes) != len(original.Routes) {
		t.Errorf("Route count mismatch: got %d, want %d", len(loaded.Routes), len(original.Routes))
	}

	if loaded.Routes[0].TxHash != original.Routes[0].TxHash {
		t.Errorf("TxHash mismatch: got %s, want %s", loaded.Routes[0].TxHash, original.Routes[0].TxHash)
	}

	if loaded.Routes[0].RouteInfo.DestinationDomain != original.Routes[0].RouteInfo.DestinationDomain {
		t.Errorf("DestinationDomain mismatch: got %d, want %d",
			loaded.Routes[0].RouteInfo.DestinationDomain,
			original.Routes[0].RouteInfo.DestinationDomain)
	}

	t.Log("✓ JSON serialization round-trip successful!")
}
