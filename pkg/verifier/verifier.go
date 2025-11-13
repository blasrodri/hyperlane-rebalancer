package verifier

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	warptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	"github.com/celestiaorg/celestia-rebalancer/pkg/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

// Verifier validates that a transaction matches the intended routes
type Verifier struct{}

// NewVerifier creates a new transaction verifier
func NewVerifier() *Verifier {
	return &Verifier{}
}

// VerifyResult contains the result of transaction verification
type VerifyResult struct {
	Valid        bool     `json:"valid"`
	MatchedCount int      `json:"matched_count"`
	TotalRoutes  int      `json:"total_routes"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// VerifyFromFiles reads routes and transaction from files and verifies them
func (v *Verifier) VerifyFromFiles(routesFile, txFile string) (*VerifyResult, error) {
	// Read routes
	routesData, err := os.ReadFile(routesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read routes file: %w", err)
	}

	var routes types.Routes
	if err := json.Unmarshal(routesData, &routes); err != nil {
		return nil, fmt.Errorf("failed to parse routes file: %w", err)
	}

	// Read transaction
	txData, err := os.ReadFile(txFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction file: %w", err)
	}

	var txRaw tx.TxRaw
	if err := json.Unmarshal(txData, &txRaw); err != nil {
		return nil, fmt.Errorf("failed to parse transaction file: %w", err)
	}

	return v.Verify(&routes, &txRaw)
}

// Verify checks if a transaction matches the intended routes
func (v *Verifier) Verify(routes *types.Routes, txRaw *tx.TxRaw) (*VerifyResult, error) {
	result := &VerifyResult{
		Valid:       true,
		TotalRoutes: len(routes.Routes),
	}

	// Decode transaction body
	var txBody tx.TxBody
	if err := txBody.Unmarshal(txRaw.BodyBytes); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to decode transaction body: %v", err))
		return result, nil
	}

	// Extract MsgRemoteTransfer messages
	var remoteTxs []*warptypes.MsgRemoteTransfer
	for _, anyMsg := range txBody.Messages {
		// Check type URL to identify MsgRemoteTransfer
		if anyMsg.TypeUrl == "/hyperlane.warp.v1.MsgRemoteTransfer" {
			var remoteMsg warptypes.MsgRemoteTransfer
			if err := remoteMsg.Unmarshal(anyMsg.Value); err != nil {
				continue
			}
			remoteTxs = append(remoteTxs, &remoteMsg)
		}
	}

	// Check if we have the right number of messages
	if len(remoteTxs) != len(routes.Routes) {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("transaction has %d MsgRemoteTransfer messages, but routes has %d entries",
				len(remoteTxs), len(routes.Routes)))
	}

	// Verify each route matches a message
	for i, route := range routes.Routes {
		if route.RouteInfo == nil {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("route %d from tx %s has no routing info", i, route.TxHash))
			continue
		}

		// Find matching message
		found := false
		for _, msg := range remoteTxs {
			if v.MatchesRoute(msg, &route) {
				found = true
				result.MatchedCount++
				break
			}
		}

		if !found {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("no matching MsgRemoteTransfer found for route %d (tx: %s, domain: %d, amount: %s)",
					i, route.TxHash, route.RouteInfo.DestinationDomain, route.Amount))
		}
	}

	return result, nil
}

// MatchesRoute checks if a MsgRemoteTransfer matches a HyperlaneRoute
func (v *Verifier) MatchesRoute(msg *warptypes.MsgRemoteTransfer, route *types.HyperlaneRoute) bool {
	// Check destination domain
	if msg.DestinationDomain != route.RouteInfo.DestinationDomain {
		return false
	}

	// Check amount
	expectedAmount := route.Amount
	if route.RouteInfo.Amount != "" {
		expectedAmount = route.RouteInfo.Amount
	}

	if msg.Amount.String() != expectedAmount {
		return false
	}

	// Check token ID (convert both to hex strings for comparison)
	msgTokenIDHex := fmt.Sprintf("%x", msg.TokenId[:])
	expectedTokenIDHex := strings.TrimPrefix(strings.ToLower(route.RouteInfo.TokenID), "0x")
	if msgTokenIDHex != expectedTokenIDHex {
		return false
	}

	// Check recipient (convert both to hex strings for comparison)
	msgRecipientHex := fmt.Sprintf("%x", msg.Recipient[:])
	expectedRecipientHex := strings.TrimPrefix(strings.ToLower(route.RouteInfo.Recipient), "0x")

	// Handle Cosmos bech32 addresses - decode and compare bytes
	if !strings.HasPrefix(route.RouteInfo.Recipient, "0x") {
		// Attempt to decode as bech32
		addr, err := sdk.AccAddressFromBech32(route.RouteInfo.Recipient)
		if err == nil {
			// Pad to 32 bytes for comparison
			paddedAddr := make([]byte, 32)
			copy(paddedAddr[32-len(addr.Bytes()):], addr.Bytes())
			expectedRecipientHex = fmt.Sprintf("%x", paddedAddr)
		}
	} else {
		// For EVM addresses, need to pad to 32 bytes
		recipientBytes, err := hex.DecodeString(expectedRecipientHex)
		if err == nil && len(recipientBytes) < 32 {
			paddedRecipient := make([]byte, 32)
			copy(paddedRecipient[32-len(recipientBytes):], recipientBytes)
			expectedRecipientHex = fmt.Sprintf("%x", paddedRecipient)
		}
	}

	if msgRecipientHex != expectedRecipientHex {
		return false
	}

	return true
}

// PrintResult prints the verification result in a human-readable format
func (v *Verifier) PrintResult(result *VerifyResult) {
	if result.Valid {
		fmt.Println("✓ Transaction verification PASSED")
		fmt.Printf("  Matched %d/%d routes\n", result.MatchedCount, result.TotalRoutes)
	} else {
		fmt.Println("✗ Transaction verification FAILED")
		fmt.Printf("  Matched %d/%d routes\n", result.MatchedCount, result.TotalRoutes)
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("  - %s\n", warn)
		}
	}
}
