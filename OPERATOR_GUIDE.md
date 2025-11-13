# Operator Guide - Celestia Rebalancer

**Quick Start**: This tool **automatically creates** all transactions for you. You just need to **review** and **sign** them.

## What This Tool Does

✅ **Parses** incoming Hyperlane transfers
✅ **Creates** unsigned transactions automatically
✅ **Verifies** everything matches
❌ You **DON'T** manually create anything - the tool does it all!

## Your Role as Operator

1. **Run** the tool commands
2. **Review** the output
3. **Sign** the transaction (with your multisig key)
4. **Broadcast** to the network

That's it! The tool handles all the complex Hyperlane message construction.

---

## Installation

### Download Binary (Recommended)

```bash
# Linux
wget https://github.com/celestiaorg/celestia-rebalancer/releases/latest/download/celestia-rebalancer-linux-amd64
chmod +x celestia-rebalancer-linux-amd64
sudo mv celestia-rebalancer-linux-amd64 /usr/local/bin/celestia-rebalancer

# macOS (ARM)
wget https://github.com/celestiaorg/celestia-rebalancer/releases/latest/download/celestia-rebalancer-darwin-arm64
chmod +x celestia-rebalancer-darwin-arm64
sudo mv celestia-rebalancer-darwin-arm64 /usr/local/bin/celestia-rebalancer

# Verify
celestia-rebalancer --help
```

### Build from Source

```bash
git clone https://github.com/celestiaorg/celestia-rebalancer.git
cd celestia-rebalancer
go build -o celestia-rebalancer ./cmd/celestia-rebalancer
```

## Prerequisites

- Access to Celestia RPC (gRPC port 9090)
- Your multisig address
- Keplr wallet OR celestia-appd for signing
- (Optional) Whitelist config file

## RPC Endpoints

| Network | gRPC Endpoint |
|---------|---------------|
| **Mainnet** | `https://rpc.celestia.org:9090` |
| **Mocha Testnet** | `https://rpc-mocha.pops.one:9090` |
| **Local Node** | `localhost:9090` |

---

## Complete Workflow (5 Steps)

### Step 1: Tool Parses Incoming Transfers

**What the tool does**: Queries blockchain, finds incoming Hyperlane transfers, extracts routing info

```bash
celestia-rebalancer parse \
  --multisig-address celestia1hyperlane7x8s... \
  --from-height 2500000 \
  --to-height 2500100 \
  --rpc-url https://rpc.celestia.org:9090 \
  --config config.json \
  --output routes.json
```

**Tool Output:**
```
Loading config from config.json...
✓ Config loaded with 2 domains configured
Parsing transactions from height 2500000 to 2500100...
Found 3 routes with total amount: 150000000 utia
Routes saved to routes.json
```

**What you do**: Review `routes.json` to verify the routes look correct

```bash
cat routes.json
```

**What to check:**
- ✓ Destination domains are expected chains
- ✓ Recipient addresses look valid
- ✓ Token IDs are correct
- ✓ Amounts match what you expect

---

### Step 2: Tool Creates Transaction (Automatic!)

**What the tool does**: Automatically creates the complete unsigned transaction with all Hyperlane messages

```bash
celestia-rebalancer generate \
  --routes routes.json \
  --multisig-address celestia1hyperlane7x8s... \
  --output unsigned-tx.json
```

**Tool Output:**
```
Generating transactions from routes.json...
Generated 3 MsgRemoteTransfer messages
Messages saved to unsigned-tx.json

Next steps:
1. Review the generated messages
2. Use 'celestia-rebalancer verify' to validate
3. Sign the transaction with your multisig key
```

**What you do**: Nothing! The tool created everything. Just check the output looks reasonable.

**Important**: The tool has now created the COMPLETE transaction. You don't need to construct anything manually!

---

### Step 3: Tool Verifies Transaction (Safety Check!)

**What the tool does**: Double-checks that the generated transaction matches the original routing intent

```bash
celestia-rebalancer verify \
  --routes routes.json \
  --transaction unsigned-tx.json
```

**Tool Output (Success):**
```
Verifying transaction against routes...

✓ Transaction verification PASSED
  Matched 3/3 routes
```

**Tool Output (Failure):**
```
Verifying transaction against routes...

✗ Transaction verification FAILED
  Matched 2/3 routes

Errors:
  - Route 2 mismatch: expected domain 2340, got 1380012617
```

**What you do**:
- If PASSED ✓ → Continue to Step 4
- If FAILED ✗ → **STOP!** Regenerate the transaction (go back to Step 2)

**Never sign a transaction that fails verification!**

---

### Step 4: You Sign the Transaction

**What you do**: Sign the unsigned transaction using your multisig key

#### Option A: Using Keplr Wallet

1. Import `unsigned-tx.json` into Keplr
2. Review the transaction details carefully
3. Sign with your multisig member key
4. Collect signatures from other multisig members
5. Combine signatures in Keplr

#### Option B: Using celestia-appd CLI

```bash
# Each multisig member signs
celestia-appd tx sign unsigned-tx.json \
  --from your-key-name \
  --multisig celestia1hyperlane7x8s... \
  --chain-id celestia \
  --output-document your-signature.json

# Share signature files with other members

# Once you have enough signatures (meet threshold), combine them
celestia-appd tx multisign unsigned-tx.json multisig-name \
  signature1.json signature2.json signature3.json \
  --chain-id celestia \
  --output-document signed-tx.json
```

**What to verify before signing:**
- Check the amounts match what you reviewed in routes.json
- Verify destination addresses are correct
- Confirm token IDs are the expected Hyperlane warp routes

---

### Step 5: You Broadcast the Transaction

**What you do**: Send the signed transaction to the Celestia network

```bash
celestia-appd tx broadcast signed-tx.json \
  --chain-id celestia \
  --node https://rpc.celestia.org:26657
```

**Output:**
```
code: 0
txhash: ABC123DEF456...
```

**Monitor the transaction:**
```bash
celestia-appd query tx ABC123DEF456...
```

**Monitor Hyperlane delivery:**
- Check Hyperlane explorer for message delivery status
- Verify funds arrived at destination chain

---

## Configuration (Recommended)

### Whitelist Config

Create `config.json` to only allow approved recipient addresses:

```json
{
  "whitelist": {
    "domains": {
      "2340": [
        "0x742d35cc6634c0532925a3b844bc9e7595f0beb0"
      ],
      "1": [
        "0x1234567890123456789012345678901234567890"
      ],
      "137": [
        "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
      ]
    }
  }
}
```

**Usage:**
```bash
cp config.example.json config.json
# Edit config.json with your approved addresses
```

**Security**: Always use a whitelist in production to prevent funds from being routed to unauthorized addresses.

---

## Common Scenarios

### Daily Rebalancing

```bash
#!/bin/bash
# daily-check.sh

MULTISIG="celestia1hyperlane7x8s..."
RPC="https://rpc.celestia.org:9090"
CONFIG="config.json"

# Get current height and calculate 24h ago
CURRENT=$(celestia-appd query block | jq -r .block.header.height)
FROM=$((CURRENT - 7200))

# Parse
./celestia-rebalancer parse \
  --multisig-address $MULTISIG \
  --from-height $FROM \
  --to-height $CURRENT \
  --rpc-url $RPC \
  --config $CONFIG \
  --output routes-$(date +%Y%m%d).json

# Check if any routes found
COUNT=$(jq '.routes | length' routes-$(date +%Y%m%d).json)
if [ "$COUNT" -gt 0 ]; then
  echo "✓ Found $COUNT routes"

  # Generate transaction
  ./celestia-rebalancer generate \
    --routes routes-$(date +%Y%m%d).json \
    --multisig-address $MULTISIG \
    --output unsigned-tx-$(date +%Y%m%d).json

  # Verify
  ./celestia-rebalancer verify \
    --routes routes-$(date +%Y%m%d).json \
    --transaction unsigned-tx-$(date +%Y%m%d).json

  if [ $? -eq 0 ]; then
    echo "✓ Verification passed! Ready to sign: unsigned-tx-$(date +%Y%m%d).json"
  else
    echo "✗ Verification failed! Do not sign!"
    exit 1
  fi
else
  echo "No routes found in last 24 hours"
fi
```

### Emergency Single Transfer

If you know the exact block where a transfer occurred:

```bash
# Parse just that block
./celestia-rebalancer parse \
  --multisig-address celestia1hyperlane7x8s... \
  --from-height 2500042 \
  --to-height 2500042 \
  --rpc-url https://rpc.celestia.org:9090 \
  --output emergency-route.json

# Generate immediately
./celestia-rebalancer generate \
  --routes emergency-route.json \
  --multisig-address celestia1hyperlane7x8s... \
  --output emergency-tx.json

# Verify and sign ASAP
./celestia-rebalancer verify \
  --routes emergency-route.json \
  --transaction emergency-tx.json
```

---

## Troubleshooting

### No routes found

```
Found 0 routes with total amount: 0
```

**Possible causes:**
- No MsgRemoteTransfer in that height range
- Wrong multisig address
- RPC connection issues

**Fix:**
```bash
# Verify multisig address
celestia-appd query bank balances celestia1hyperlane7x8s...

# Try wider range
--from-height 2000000 --to-height 3000000

# Test RPC
telnet rpc.celestia.org 9090
```

### Whitelist rejection

```
Warning: tx ABC123 failed whitelist validation: recipient 0x999... is not whitelisted for domain 2340
```

**Fix**: Add the address to your `config.json`:

```json
{
  "whitelist": {
    "domains": {
      "2340": [
        "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
        "0x999..."  // Add new approved address
      ]
    }
  }
}
```

### Verification fails

```
✗ Transaction verification FAILED
```

**Fix**:
1. Read the error message carefully
2. Check `routes.json` and `unsigned-tx.json` are from the same run
3. Regenerate: `./celestia-rebalancer generate ...`
4. Verify again
5. **Never sign a failed verification!**

### Token ID errors

```
Error: token ID must be exactly 32 bytes, got 16 bytes
```

**Fix**: Token IDs must be 64 hex characters (32 bytes):
```json
{
  "token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
}
```

---

## Security Best Practices

1. ✅ **Always use whitelist** in production (`--config config.json`)
2. ✅ **Always verify** before signing
3. ✅ **Review routes.json** manually before generating
4. ✅ **Use hardware wallets** for multisig keys
5. ✅ **Test on testnet first** with small amounts
6. ✅ **Keep logs** of all operations
7. ✅ **Monitor Hyperlane delivery** after broadcast
8. ✅ **Regular audits** of multisig balance

---

## What You Need to Know

### Custom Hook Metadata Format

Users send funds via Hyperlane `MsgRemoteTransfer` with routing info in `custom_hook_metadata`:

```json
{
  "destination_domain": 2340,
  "recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
  "amount": "10000000"
}
```

**You don't need to construct this** - the tool extracts it automatically from incoming transfers!

### Common Domain IDs

| Chain | Domain ID |
|-------|-----------|
| Ethereum | 1 |
| Polygon | 137 |
| Arbitrum | 42161 |
| Optimism | 10 |
| Eden | 2340 |
| Celestia | 69420 |

---

## Support

- **GitHub Issues**: https://github.com/celestiaorg/celestia-rebalancer/issues
- **Documentation**: See other `.md` files in this repo
- **Celestia Discord**: https://discord.gg/celestiacommunity

---

**Remember**: This tool does all the hard work. You just review, sign, and broadcast! 🚀

**Version**: 1.1.0
**Last Updated**: 2025-11-13
