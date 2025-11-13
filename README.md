# Celestia Rebalancer

A CLI tool for managing Hyperlane multisig rebalancing on Celestia.

## Overview

This tool helps automate the **Hyperlane multi-hop fund forwarding** process by:

1. **Parsing** incoming `MsgRemoteTransfer` transactions to extract routing information from `custom_hook_metadata`
2. **Generating** multisig transactions with Hyperlane `MsgRemoteTransfer` messages
3. **Verifying** that transactions match the intended routes before signing
4. **Validating** recipient addresses against a configurable whitelist for security

## Use Case: Hyperlane Multi-Hop Forwarding

When funds arrive at a Celestia multisig via Hyperlane from another chain (e.g., Noble), they need to be forwarded to their final destination (e.g., Eden). The routing information is embedded in the incoming `MsgRemoteTransfer`'s `custom_hook_metadata` field.

**Flow**: Noble (Hyperlane) → **Celestia Multisig** → Eden (Hyperlane)

This tool:
- Monitors incoming Hyperlane transfers to the multisig
- Extracts the routing details (destination domain, recipient address, token ID)
- Creates the appropriate Hyperlane cross-chain transfer messages
- Validates recipient addresses against a whitelist
- Ensures the transaction matches the original intent before signing

## Installation

### From Source

```bash
git clone https://github.com/celestiaorg/celestia-rebalancer.git
cd celestia-rebalancer
go build -o celestia-rebalancer ./cmd/celestia-rebalancer
```

### Verify Installation

```bash
./celestia-rebalancer --help
```

## Configuration

### Whitelist Config (Recommended for Production)

Create a `config.json` file to whitelist allowed recipient addresses per destination domain:

```json
{
  "whitelist": {
    "domains": {
      "2340": [
        "0x742d35cc6634c0532925a3b844bc9e7595f0beb0"
      ],
      "1": [
        "0x1234567890123456789012345678901234567890"
      ]
    }
  }
}
```

Copy the example:
```bash
cp config.example.json config.json
# Edit config.json with your whitelisted addresses
```

**Security Note**: Without a config file, any recipient address will be accepted. For production deployments, always use a whitelist.

## Custom Hook Metadata Format

Incoming `MsgRemoteTransfer` transactions must include routing information in the `custom_hook_metadata` field:

```json
{
  "destination_domain": 2340,
  "recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
  "amount": "10000000"
}
```

**Fields:**
- `destination_domain` (required): Hyperlane domain ID of the final destination chain
- `recipient` (required): Final destination address (EVM hex or Cosmos bech32)
- `token_id` (required): Hyperlane warp route token ID (must be 32 bytes hex)
- `amount` (optional): Amount to forward (defaults to received amount)

## Operator Workflow

### Step 1: Parse Incoming Transfers

Query the blockchain for incoming `MsgRemoteTransfer` transactions and extract routing information:

```bash
./celestia-rebalancer parse \
  --multisig-address celestia1hyperlane7x8s... \
  --from-height 2500000 \
  --to-height 2500100 \
  --rpc-url https://rpc.celestia.org:9090 \
  --config config.json \
  --output routes.json
```

**What this does:**
- Connects to Celestia gRPC endpoint
- Queries blocks from height 2,500,000 to 2,500,100
- Filters for `MsgRemoteTransfer` transactions sent to the multisig
- Extracts `custom_hook_metadata` from each transfer
- Validates recipient addresses against whitelist (if config provided)
- Outputs results to `routes.json`

**Output Example:**
```
Loading config from config.json...
✓ Config loaded with 2 domains configured
Parsing transactions from height 2500000 to 2500100...
Found 3 routes with total amount: 150000000
Routes saved to routes.json
```

**Review the output:**
```bash
cat routes.json
```

```json
{
  "routes": [
    {
      "tx_hash": "ABC123...",
      "block_height": 2500042,
      "from": "noble1user...",
      "amount": "50000000",
      "denom": "utia",
      "custom_hook_metadata": "{\"destination_domain\": 2340, \"recipient\": \"0x742d35...\", \"token_id\": \"0x1234...\"}",
      "route_info": {
        "destination_domain": 2340,
        "recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
        "token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
      }
    }
  ],
  "total_amount": "50000000",
  "multisig_address": "celestia1hyperlane7x8s..."
}
```

### Step 2: Generate Multisig Transaction

Create unsigned `MsgRemoteTransfer` messages from the parsed routes:

```bash
./celestia-rebalancer generate \
  --routes routes.json \
  --multisig-address celestia1hyperlane7x8s... \
  --output unsigned-tx.json
```

**What this does:**
- Reads `routes.json`
- Creates one `MsgRemoteTransfer` message per route
- Sets sender as the multisig address
- Uses destination domain, recipient, token ID, and amount from routing info
- Outputs unsigned transaction messages

**Output:**
```
Generating transactions from routes.json...
Generated 3 MsgRemoteTransfer messages
Messages saved to unsigned-tx.json

Next steps:
1. Review the generated messages
2. Use 'celestia-rebalancer verify' to validate
3. Create multisig transaction using celestia-appd or Keplr
```

### Step 3: Verify Transaction

Validate that the generated transaction matches the intended routes:

```bash
./celestia-rebalancer verify \
  --routes routes.json \
  --transaction unsigned-tx.json
```

**What this does:**
- Compares each `MsgRemoteTransfer` in `unsigned-tx.json` with routes in `routes.json`
- Checks: destination domain, amount, token ID (byte-by-byte), recipient address
- Reports any mismatches

**Output (Success):**
```
Verifying transaction against routes...

✓ Transaction verification PASSED
  Matched 3/3 routes
```

**Output (Failure):**
```
Verifying transaction against routes...

✗ Transaction verification FAILED
  Matched 2/3 routes

Errors:
  - no matching MsgRemoteTransfer found for route 2 (tx: XYZ789, domain: 2340, amount: 50000000)
```

**If verification fails:** Regenerate the transaction and verify again. Do NOT proceed to signing.

### Step 4: Sign and Broadcast

Use Keplr wallet or `celestia-appd` multisig to sign and broadcast:

#### Option A: Using Keplr

1. Import `unsigned-tx.json` into Keplr
2. Collect signatures from multisig members
3. Broadcast the signed transaction

#### Option B: Using celestia-appd

```bash
# Each signer creates a signature
celestia-appd tx sign unsigned-tx.json \
  --from signer1 \
  --multisig celestia1hyperlane7x8s... \
  --output-document signer1.json

# Combine signatures (requires threshold number of signers)
celestia-appd tx multisign unsigned-tx.json multisig-name \
  signer1.json signer2.json signer3.json \
  --output-document signed-tx.json

# Broadcast
celestia-appd tx broadcast signed-tx.json
```

### Step 5: Monitor Delivery

The Hyperlane relayers will automatically deliver the funds to the destination chain. Monitor:
- Transaction status on Celestia
- Hyperlane message delivery (check Hyperlane explorer)
- Funds arrival on destination chain

## RPC Endpoints

| Network | gRPC Endpoint | Use Case |
|---------|---------------|----------|
| **Mainnet** | `https://rpc.celestia.org:9090` | Production |
| **Mocha Testnet** | `https://rpc-mocha.pops.one:9090` | Testing |
| **Local Node** | `localhost:9090` | Development |

## Common Hyperlane Domain IDs

| Chain      | Domain ID   | Notes |
|------------|-------------|-------|
| Ethereum   | 1           | Mainnet |
| Polygon    | 137         | PoS chain |
| Avalanche  | 43114       | C-Chain |
| BSC        | 56          | Binance Smart Chain |
| Arbitrum   | 42161       | L2 |
| Optimism   | 10          | L2 |
| Celestia   | 69420       | Celestia mainnet |
| Eden       | 2340        | Eden network |

## Troubleshooting

### Issue: No routes found

```
Found 0 routes with total amount: 0
```

**Possible causes:**
- No `MsgRemoteTransfer` transactions in the specified height range
- Wrong multisig address
- Incorrect RPC endpoint

**Solution:**
```bash
# Verify multisig address
celestia-appd query bank balances celestia1hyperlane7x8s...

# Try wider height range
--from-height 2000000 --to-height 2600000

# Check RPC connectivity
telnet rpc.celestia.org 9090
```

### Issue: Routes rejected by whitelist

```
Warning: tx ABC123 failed whitelist validation: recipient 0x999... is not whitelisted for domain 2340
```

**Solution:**
- Add the recipient address to `config.json` under the appropriate domain
- OR remove `--config` flag to disable whitelist (NOT recommended for production)

```json
{
  "whitelist": {
    "domains": {
      "2340": [
        "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
        "0x999..."  // Add new address
      ]
    }
  }
}
```

### Issue: Transaction verification failed

```
✗ Transaction verification FAILED
```

**Solution:**
1. Check the error message for specific field mismatches
2. Regenerate the transaction: `./celestia-rebalancer generate ...`
3. Verify again before signing
4. If still failing, check that `routes.json` and `unsigned-tx.json` are from the same run

### Issue: Token ID must be 32 bytes

```
Error: token ID must be exactly 32 bytes, got 16 bytes
```

**Solution:**
Token IDs in `custom_hook_metadata` must be 32-byte hex strings (64 hex characters):
```json
{
  "token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
}
```

## Operational Best Practices

### Security

1. **Always use whitelist in production**: `--config config.json`
2. **Always verify before signing**: Run `verify` command
3. **Use hardware wallets** for multisig signers
4. **Keep logs** of all rebalancing operations
5. **Test on testnet first** with small amounts
6. **Set up alerts** for large transfers
7. **Regular audits** of multisig balance

### Automation

Consider creating a script for regular operations:

```bash
#!/bin/bash
# daily-rebalance.sh

MULTISIG="celestia1hyperlane7x8s..."
RPC="https://rpc.celestia.org:9090"
CONFIG="config.json"

# Get current height
CURRENT_HEIGHT=$(celestia-appd query block | jq -r .block.header.height)
FROM_HEIGHT=$((CURRENT_HEIGHT - 7200))  # ~24 hours ago

# Parse
./celestia-rebalancer parse \
  --multisig-address $MULTISIG \
  --from-height $FROM_HEIGHT \
  --to-height $CURRENT_HEIGHT \
  --rpc-url $RPC \
  --config $CONFIG \
  --output "routes-$(date +%Y%m%d).json"

# Check if any routes found
ROUTE_COUNT=$(jq '.routes | length' "routes-$(date +%Y%m%d).json")
if [ "$ROUTE_COUNT" -gt 0 ]; then
  echo "Found $ROUTE_COUNT routes, generating transaction..."

  ./celestia-rebalancer generate \
    --routes "routes-$(date +%Y%m%d).json" \
    --multisig-address $MULTISIG \
    --output "unsigned-tx-$(date +%Y%m%d).json"

  ./celestia-rebalancer verify \
    --routes "routes-$(date +%Y%m%d).json" \
    --transaction "unsigned-tx-$(date +%Y%m%d).json"

  echo "✓ Transaction ready for signing: unsigned-tx-$(date +%Y%m%d).json"
else
  echo "No routes found in the last 24 hours"
fi
```

### Monitoring

Monitor your multisig address:
```bash
# Check balance
celestia-appd query bank balances celestia1hyperlane7x8s...

# Watch for incoming transfers (in real-time)
# TODO: Implement watch command
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Hyperlane Multi-Hop Flow                 │
└─────────────────────────────────────────────────────────────┘

   Noble                 Celestia              Eden
     │                      │                   │
     │  MsgRemoteTransfer   │                   │
     │  (custom_hook_       │                   │
     │   metadata with      │                   │
     │   routing info)      │                   │
     ├─────────────────────►│                   │
     │                      │                   │
     │               ┌──────▼─────────┐         │
     │               │   Multisig     │         │
     │               │   receives     │         │
     │               │   funds        │         │
     │               └──────┬─────────┘         │
     │                      │                   │
     │               ┌──────▼─────────┐         │
     │               │ celestia-      │         │
     │               │ rebalancer     │         │
     │               │ parse          │         │
     │               └──────┬─────────┘         │
     │                      │                   │
     │               [routes.json]              │
     │                      │                   │
     │               ┌──────▼─────────┐         │
     │               │ celestia-      │         │
     │               │ rebalancer     │         │
     │               │ generate       │         │
     │               └──────┬─────────┘         │
     │                      │                   │
     │               [unsigned-tx.json]         │
     │                      │                   │
     │               ┌──────▼─────────┐         │
     │               │ celestia-      │         │
     │               │ rebalancer     │         │
     │               │ verify         │         │
     │               └──────┬─────────┘         │
     │                      │                   │
     │               ┌──────▼─────────┐         │
     │               │ Keplr/appd     │         │
     │               │ multisig sign  │         │
     │               └──────┬─────────┘         │
     │                      │                   │
     │                      │  MsgRemoteTransfer│
     │                      ├──────────────────►│
     │                      │                   │
     │                      │    Hyperlane      │
     │                      │    relayers       │
     │                      │    deliver        │
     │                      │                   ▼
                                           [Eden receives
                                            funds]
```

## Development

### Run Tests

```bash
go test ./pkg/... -v
```

All 29 tests should pass:
- ✓ Integration tests (5)
- ✓ Types tests (6)
- ✓ Generator tests (5)
- ✓ Verifier tests (3)
- ✓ Config tests (3)

### Build

```bash
go build -o celestia-rebalancer ./cmd/celestia-rebalancer
```

### Format

```bash
go fmt ./...
```

## Documentation

- **Operator Guide**: See `OPERATOR_GUIDE.md` for detailed step-by-step instructions
- **Technical Details**: See `HYPERLANE_MULTIHOP.md` for implementation details
- **Testing**: See `TESTING.md` for testing guide
- **Updates**: See `CUSTOM_HOOK_METADATA_UPDATE.md` for recent changes

## Support

- **GitHub Issues**: https://github.com/celestiaorg/celestia-rebalancer/issues
- **Celestia Discord**: https://discord.gg/celestiacommunity

## License

Apache 2.0 (same as Celestia)

---

**Version**: 1.1.0
**Status**: ✅ Production Ready
**Last Updated**: 2025-11-13
