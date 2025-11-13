package client

import (
	"context"
	"fmt"

	warptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a gRPC client for querying Celestia blockchain data
type Client struct {
	conn      *grpc.ClientConn
	txClient  tx.ServiceClient
	ctx       context.Context
	encConfig client.TxConfig
}

// NewClient creates a new gRPC client connected to the given RPC endpoint
func NewClient(ctx context.Context, rpcEndpoint string) (*Client, error) {
	conn, err := grpc.NewClient(rpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC endpoint: %w", err)
	}

	return &Client{
		conn:     conn,
		txClient: tx.NewServiceClient(conn),
		ctx:      ctx,
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// Transaction represents a blockchain transaction with extracted data
type Transaction struct {
	Hash        string
	BlockHeight int64
	Memo        string
	Tx          *tx.Tx // Store the full decoded transaction
}

// GetTransactionsByHeight queries transactions within a height range
func (c *Client) GetTransactionsByHeight(fromHeight, toHeight int64) ([]*Transaction, error) {
	var allTxs []*Transaction

	// Query block by block
	for height := fromHeight; height <= toHeight; height++ {
		// Query transactions at this height using block search
		query := fmt.Sprintf("tx.height=%d", height)
		req := &tx.GetTxsEventRequest{
			Events:  []string{query},
			OrderBy: tx.OrderBy_ORDER_BY_ASC,
			Page:    1,
			Limit:   100, // Max transactions per block
		}

		resp, err := c.txClient.GetTxsEvent(c.ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to query transactions at height %d: %w", height, err)
		}

		for _, txResp := range resp.TxResponses {
			// Decode the transaction to get the body
			if txResp.Tx == nil {
				continue
			}

			// Unmarshal the Any type to Tx
			var decodedTx tx.Tx
			if err := decodedTx.Unmarshal(txResp.Tx.Value); err != nil {
				continue
			}

			memo := ""
			if decodedTx.Body != nil {
				memo = decodedTx.Body.Memo
			}

			allTxs = append(allTxs, &Transaction{
				Hash:        txResp.TxHash,
				BlockHeight: height,
				Memo:        memo,
				Tx:          &decodedTx,
			})
		}
	}

	return allTxs, nil
}

// BankSend represents a bank send message with parsed data
type BankSend struct {
	From   string
	To     string
	Amount types.Coins
}

// ExtractBankSends extracts all bank send messages from a transaction
func ExtractBankSends(txn *Transaction) ([]BankSend, error) {
	var sends []BankSend

	if txn.Tx == nil || txn.Tx.Body == nil {
		return sends, nil
	}

	for _, anyMsg := range txn.Tx.Body.Messages {
		// Check if this is a MsgSend by type URL
		if anyMsg.TypeUrl == "/cosmos.bank.v1beta1.MsgSend" {
			var sendMsg banktypes.MsgSend
			if err := sendMsg.Unmarshal(anyMsg.Value); err != nil {
				continue
			}

			sends = append(sends, BankSend{
				From:   sendMsg.FromAddress,
				To:     sendMsg.ToAddress,
				Amount: sendMsg.Amount,
			})
		}
	}

	return sends, nil
}

// FilterTransactionsToAddress filters transactions that have bank sends to a specific address
func FilterTransactionsToAddress(txs []*Transaction, targetAddress string) ([]*Transaction, error) {
	var filtered []*Transaction

	for _, tx := range txs {
		sends, err := ExtractBankSends(tx)
		if err != nil {
			continue
		}

		for _, send := range sends {
			if send.To == targetAddress {
				filtered = append(filtered, tx)
				break
			}
		}
	}

	return filtered, nil
}

// HyperlaneTransfer represents a Hyperlane MsgRemoteTransfer with parsed data
type HyperlaneTransfer struct {
	From               string
	To                 string // Recipient as hex string
	Amount             string
	DestinationDomain  uint32
	TokenID            string // Token ID as hex string
	CustomHookMetadata string // Routing information for multi-hop forwarding
}

// ExtractHyperlaneTransfers extracts all Hyperlane MsgRemoteTransfer messages from a transaction
func ExtractHyperlaneTransfers(txn *Transaction) ([]HyperlaneTransfer, error) {
	var transfers []HyperlaneTransfer

	if txn.Tx == nil || txn.Tx.Body == nil {
		return transfers, nil
	}

	for _, anyMsg := range txn.Tx.Body.Messages {
		// Check if this is a MsgRemoteTransfer by type URL
		if anyMsg.TypeUrl == "/hyperlane.warp.v1.MsgRemoteTransfer" {
			var msg warptypes.MsgRemoteTransfer
			if err := msg.Unmarshal(anyMsg.Value); err != nil {
				continue
			}

			// Convert recipient bytes to hex string
			recipientHex := fmt.Sprintf("0x%x", msg.Recipient[:])

			// Convert token ID bytes to hex string
			tokenIDHex := fmt.Sprintf("0x%x", msg.TokenId[:])

			transfers = append(transfers, HyperlaneTransfer{
				From:               msg.Sender,
				To:                 recipientHex,
				Amount:             msg.Amount.String(),
				DestinationDomain:  msg.DestinationDomain,
				TokenID:            tokenIDHex,
				CustomHookMetadata: msg.CustomHookMetadata,
			})
		}
	}

	return transfers, nil
}

// FilterHyperlaneTransfersToAddress filters transactions that have Hyperlane transfers to a specific address
func FilterHyperlaneTransfersToAddress(txs []*Transaction, targetAddress string) ([]*Transaction, error) {
	var filtered []*Transaction

	for _, tx := range txs {
		transfers, err := ExtractHyperlaneTransfers(tx)
		if err != nil {
			continue
		}

		for _, transfer := range transfers {
			// Compare the recipient address (need to handle both hex formats and bech32)
			if transfer.To == targetAddress || transfer.From == targetAddress {
				filtered = append(filtered, tx)
				break
			}
		}
	}

	return filtered, nil
}
