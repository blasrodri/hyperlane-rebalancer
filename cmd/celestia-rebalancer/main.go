package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/celestiaorg/celestia-rebalancer/pkg/generator"
	"github.com/celestiaorg/celestia-rebalancer/pkg/parser"
	"github.com/celestiaorg/celestia-rebalancer/pkg/types"
	"github.com/celestiaorg/celestia-rebalancer/pkg/verifier"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "celestia-rebalancer",
		Short: "CLI tool for managing Hyperlane multisig rebalancing on Celestia",
		Long: `celestia-rebalancer helps automate the Hyperlane rebalancing process by:
  1. Parsing incoming transactions to extract routing information
  2. Generating multisig transactions for Hyperlane MsgRemoteTransfer
  3. Verifying that transactions match the intended routes`,
	}

	rootCmd.AddCommand(
		parseCmd(),
		generateCmd(),
		verifyCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseCmd() *cobra.Command {
	var (
		multisigAddr string
		fromHeight   int64
		toHeight     int64
		rpcURL       string
		outputFile   string
		configFile   string
	)

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse incoming MsgRemoteTransfer transactions and extract Hyperlane routing information",
		Long: `Parse MsgRemoteTransfer transactions sent to the multisig address and extract routing information from custom_hook_metadata.

The custom_hook_metadata should be JSON format:
{
  "destination_domain": 1380012617,
  "recipient": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "token_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
  "amount": "10000"
}

Optional config file for address whitelisting:
{
  "whitelist": {
    "domains": {
      "2340": ["0x742d35cc6634c0532925a3b844bc9e7595f0beb0"],
      "1": ["0x1234567890123456789012345678901234567890"]
    }
  }
}`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config if provided
			var config *types.Config
			var err error
			if configFile != "" {
				fmt.Printf("Loading config from %s...\n", configFile)
				config, err = types.LoadConfig(configFile)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
				fmt.Printf("✓ Config loaded with %d domains configured\n", len(config.Whitelist.Domains))
			}

			// Create parser with or without config
			var p *parser.Parser
			if config != nil {
				p, err = parser.NewParserWithConfig(rpcURL, config)
			} else {
				p, err = parser.NewParser(rpcURL)
			}
			if err != nil {
				return fmt.Errorf("failed to create parser: %w", err)
			}
			defer p.Close()

			// Parse routes
			fmt.Printf("Parsing transactions from height %d to %d...\n", fromHeight, toHeight)
			routes, err := p.ParseRoutes(multisigAddr, fromHeight, toHeight)
			if err != nil {
				return fmt.Errorf("failed to parse routes: %w", err)
			}

			fmt.Printf("Found %d routes with total amount: %s\n", len(routes.Routes), routes.TotalAmount)

			// Output results
			data, err := json.MarshalIndent(routes, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal routes: %w", err)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, data, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Printf("Routes saved to %s\n", outputFile)
			} else {
				fmt.Println(string(data))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&multisigAddr, "multisig-address", "", "Multisig address to filter transactions (required)")
	cmd.Flags().Int64Var(&fromHeight, "from-height", 0, "Starting block height (required)")
	cmd.Flags().Int64Var(&toHeight, "to-height", 0, "Ending block height (required)")
	cmd.Flags().StringVar(&rpcURL, "rpc-url", "localhost:9090", "gRPC endpoint URL")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "routes.json", "Output file for routes")
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Optional config file for address whitelisting")

	cmd.MarkFlagRequired("multisig-address")
	cmd.MarkFlagRequired("from-height")
	cmd.MarkFlagRequired("to-height")

	return cmd
}

func generateCmd() *cobra.Command {
	var (
		routesFile   string
		multisigAddr string
		outputFile   string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate unsigned multisig transaction from routes",
		Long:  `Generate unsigned Hyperlane MsgRemoteTransfer transactions from parsed routes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create generator
			gen := generator.NewGenerator(multisigAddr)

			// Generate messages
			fmt.Printf("Generating transactions from %s...\n", routesFile)
			msgs, err := gen.GenerateFromFile(routesFile)
			if err != nil {
				return fmt.Errorf("failed to generate transactions: %w", err)
			}

			fmt.Printf("Generated %d MsgRemoteTransfer messages\n", len(msgs))

			// Output transaction
			// Note: This outputs the messages in JSON format
			// For actual signing, you'll need to use celestia-appd or Keplr
			data, err := json.MarshalIndent(msgs, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal messages: %w", err)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, data, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Printf("Messages saved to %s\n", outputFile)
			} else {
				fmt.Println(string(data))
			}

			fmt.Println("\nNext steps:")
			fmt.Println("1. Review the generated messages")
			fmt.Println("2. Use 'celestia-rebalancer verify' to validate")
			fmt.Println("3. Create multisig transaction using celestia-appd or Keplr")

			return nil
		},
	}

	cmd.Flags().StringVar(&routesFile, "routes", "routes.json", "Input routes file")
	cmd.Flags().StringVar(&multisigAddr, "multisig-address", "", "Multisig address (sender) (required)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "unsigned-tx.json", "Output file for unsigned transaction")

	cmd.MarkFlagRequired("multisig-address")

	return cmd
}

func verifyCmd() *cobra.Command {
	var (
		routesFile string
		txFile     string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify that a transaction matches the intended routes",
		Long:  `Verify that a multisig transaction contains the correct MsgRemoteTransfer messages matching the parsed routes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create verifier
			v := verifier.NewVerifier()

			// Verify
			fmt.Printf("Verifying transaction against routes...\n\n")
			result, err := v.VerifyFromFiles(routesFile, txFile)
			if err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}

			// Print result
			v.PrintResult(result)

			if !result.Valid {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&routesFile, "routes", "routes.json", "Routes file to verify against")
	cmd.Flags().StringVar(&txFile, "transaction", "unsigned-tx.json", "Transaction file to verify")

	return cmd
}
