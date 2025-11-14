package parser

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/celestiaorg/celestia-rebalancer/pkg/client"
	"github.com/celestiaorg/celestia-rebalancer/pkg/types"
)

// Parser handles parsing of transactions to extract Hyperlane routing information
type Parser struct {
	client *client.Client
	config *types.Config // Optional whitelist config
}

// NewParser creates a new parser with the given gRPC client
func NewParser(rpcEndpoint string) (*Parser, error) {
	c, err := client.NewClient(context.Background(), rpcEndpoint)
	if err != nil {
		return nil, err
	}

	return &Parser{
		client: c,
	}, nil
}

// NewParserWithConfig creates a new parser with whitelist validation enabled
func NewParserWithConfig(rpcEndpoint string, config *types.Config) (*Parser, error) {
	c, err := client.NewClient(context.Background(), rpcEndpoint)
	if err != nil {
		return nil, err
	}

	return &Parser{
		client: c,
		config: config,
	}, nil
}

// Close closes the underlying client connection
func (p *Parser) Close() error {
	return p.client.Close()
}

// ParseRoutes extracts Hyperlane routing information from MsgRemoteTransfer transactions sent to the multisig
func (p *Parser) ParseRoutes(multisigAddr string, fromHeight, toHeight int64) (*types.Routes, error) {
	// Query all transactions in the height range
	txs, err := p.client.GetTransactionsByHeight(fromHeight, toHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}

	// Filter to only transactions with Hyperlane transfers to the multisig
	filtered, err := client.FilterHyperlaneTransfersToAddress(txs, multisigAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to filter transactions: %w", err)
	}

	var routes []types.HyperlaneRoute
	totalAmount := math.ZeroInt()

	for _, tx := range filtered {
		// Extract Hyperlane transfers from the transaction
		transfers, err := client.ExtractHyperlaneTransfers(tx)
		if err != nil {
			continue
		}

		// Process each Hyperlane transfer
		for _, transfer := range transfers {
			// Skip if the transfer is not to the multisig (sender must match)
			if transfer.From != multisigAddr {
				continue
			}

			var routeInfo *types.RouteInfo

			// Check if we have custom_hook_metadata (for MsgRemoteTransfer)
			if transfer.CustomHookMetadata != "" {
				// Parse the custom_hook_metadata for routing information
				var err error
				routeInfo, err = types.ParseCustomHookMetadata(transfer.CustomHookMetadata)
				if err != nil {
					// Skip transactions without valid routing info
					fmt.Printf("Warning: tx %s has invalid custom_hook_metadata: %v\n", tx.Hash, err)
					continue
				}
			} else if transfer.DestinationDomain != 0 {
				// For bank sends with routing metadata in memo, create RouteInfo from transfer fields
				routeInfo = &types.RouteInfo{
					DestinationDomain: transfer.DestinationDomain,
					Recipient:         transfer.To,
					TokenID:           transfer.TokenID,
				}
			} else {
				// No routing information available
				fmt.Printf("Warning: tx %s has no routing information, skipping\n", tx.Hash)
				continue
			}

			// Validate against whitelist if config is provided
			if p.config != nil {
				if err := p.config.ValidateRoute(routeInfo); err != nil {
					fmt.Printf("Warning: tx %s failed whitelist validation: %v\n", tx.Hash, err)
					continue
				}
			}

			// Use the amount from the route info if specified, otherwise use the transfer amount
			amount := transfer.Amount
			if routeInfo.Amount != "" {
				amount = routeInfo.Amount
			}

			route := types.HyperlaneRoute{
				TxHash:             tx.Hash,
				BlockHeight:        tx.BlockHeight,
				From:               transfer.From,
				Amount:             amount,
				Denom:              "utia", // Hyperlane transfers use native token
				CustomHookMetadata: transfer.CustomHookMetadata,
				RouteInfo:          routeInfo,
			}

			routes = append(routes, route)

			// Add to total
			amountInt, ok := math.NewIntFromString(amount)
			if ok {
				totalAmount = totalAmount.Add(amountInt)
			}
		}
	}

	return &types.Routes{
		Routes:       routes,
		TotalAmount:  totalAmount.String(),
		MultisigAddr: multisigAddr,
	}, nil
}
