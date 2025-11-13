package generator

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cosmossdk.io/math"
	"github.com/bcp-innovations/hyperlane-cosmos/util"
	warptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	"github.com/celestiaorg/celestia-rebalancer/pkg/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Generator creates Hyperlane MsgRemoteTransfer transactions from routes
type Generator struct {
	multisigAddr string
}

// NewGenerator creates a new transaction generator
func NewGenerator(multisigAddr string) *Generator {
	return &Generator{
		multisigAddr: multisigAddr,
	}
}

// GenerateFromFile reads routes from a JSON file and generates unsigned transactions
func (g *Generator) GenerateFromFile(routesFile string) ([]sdk.Msg, error) {
	// Read routes file
	data, err := os.ReadFile(routesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read routes file: %w", err)
	}

	var routes types.Routes
	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("failed to parse routes file: %w", err)
	}

	return g.Generate(&routes)
}

// Generate creates MsgRemoteTransfer messages from parsed routes
func (g *Generator) Generate(routes *types.Routes) ([]sdk.Msg, error) {
	var msgs []sdk.Msg

	for _, route := range routes.Routes {
		if route.RouteInfo == nil {
			return nil, fmt.Errorf("route from tx %s has no routing info", route.TxHash)
		}

		// Parse amount
		amount, ok := math.NewIntFromString(route.Amount)
		if !ok {
			return nil, fmt.Errorf("invalid amount %s in route from tx %s", route.Amount, route.TxHash)
		}

		// Parse token ID
		tokenID, err := parseTokenID(route.RouteInfo.TokenID)
		if err != nil {
			return nil, fmt.Errorf("invalid token_id in route from tx %s: %w", route.TxHash, err)
		}

		// Parse and pad recipient address
		recipient, err := parseAndPadAddress(route.RouteInfo.Recipient)
		if err != nil {
			return nil, fmt.Errorf("invalid recipient address in route from tx %s: %w", route.TxHash, err)
		}

		// Create MsgRemoteTransfer
		msg := &warptypes.MsgRemoteTransfer{
			Sender:            g.multisigAddr,
			TokenId:           tokenID,
			DestinationDomain: route.RouteInfo.DestinationDomain,
			Recipient:         recipient,
			Amount:            amount,
		}

		msgs = append(msgs, msg)
	}

	return msgs, nil
}

// parseTokenID converts a token ID string (hex or bech32) to util.HexAddress
// Token IDs must be exactly 32 bytes for Hyperlane
func parseTokenID(tokenIDStr string) (util.HexAddress, error) {
	// Remove 0x prefix if present
	tokenIDStr = strings.TrimPrefix(tokenIDStr, "0x")

	// Decode hex string
	tokenID, err := hex.DecodeString(tokenIDStr)
	if err != nil {
		return util.HexAddress{}, fmt.Errorf("failed to decode token ID as hex: %w", err)
	}

	// Validate that token ID is exactly 32 bytes
	if len(tokenID) != 32 {
		return util.HexAddress{}, fmt.Errorf("token ID must be exactly 32 bytes, got %d bytes", len(tokenID))
	}

	return util.HexAddress(tokenID), nil
}

// parseAndPadAddress parses an address and pads it to 32 bytes for Hyperlane
// Supports both EVM addresses (0x...) and Cosmos bech32 addresses
func parseAndPadAddress(addrStr string) (util.HexAddress, error) {
	var addrBytes []byte

	if strings.HasPrefix(addrStr, "0x") {
		// EVM address - decode hex
		addrStr = strings.TrimPrefix(addrStr, "0x")
		decoded, err := hex.DecodeString(addrStr)
		if err != nil {
			return util.HexAddress{}, fmt.Errorf("failed to decode EVM address: %w", err)
		}
		addrBytes = decoded
	} else {
		// Assume Cosmos bech32 address
		addr, err := sdk.AccAddressFromBech32(addrStr)
		if err != nil {
			return util.HexAddress{}, fmt.Errorf("failed to decode bech32 address: %w", err)
		}
		addrBytes = addr.Bytes()
	}

	// Pad to 32 bytes (Hyperlane requirement)
	// Cosmos addresses are 20 bytes, so we left-pad with 12 zero bytes
	paddedAddr := make([]byte, 32)
	copy(paddedAddr[32-len(addrBytes):], addrBytes)

	return util.HexAddress(paddedAddr), nil
}
