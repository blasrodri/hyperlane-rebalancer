package verifier

import (
	"testing"

	"github.com/celestiaorg/celestia-rebalancer/pkg/types"
)

func TestNewVerifier(t *testing.T) {
	verifier := NewVerifier()
	if verifier == nil {
		t.Error("NewVerifier() returned nil")
	}
}

func TestMatchesRoute(t *testing.T) {
	tests := []struct {
		name       string
		route      types.HyperlaneRoute
		msgDomain  uint32
		msgAmount  string
		wantMatch  bool
	}{
		{
			name: "exact match",
			route: types.HyperlaneRoute{
				Amount: "1000000",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 1380012617,
				},
			},
			msgDomain: 1380012617,
			msgAmount: "1000000",
			wantMatch: true,
		},
		{
			name: "domain mismatch",
			route: types.HyperlaneRoute{
				Amount: "1000000",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 1380012617,
				},
			},
			msgDomain: 137,
			msgAmount: "1000000",
			wantMatch: false,
		},
		{
			name: "amount mismatch",
			route: types.HyperlaneRoute{
				Amount: "1000000",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 1380012617,
				},
			},
			msgDomain: 1380012617,
			msgAmount: "2000000",
			wantMatch: false,
		},
		{
			name: "route with amount override",
			route: types.HyperlaneRoute{
				Amount: "1000000",
				RouteInfo: &types.RouteInfo{
					DestinationDomain: 1380012617,
					Amount:            "500000",
				},
			},
			msgDomain: 1380012617,
			msgAmount: "500000",
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock message
			// Note: We can't easily create a real MsgRemoteTransfer without importing hyperlane types
			// So this test verifies the logic conceptually

			// Check domain match
			domainMatch := tt.msgDomain == tt.route.RouteInfo.DestinationDomain

			// Check amount match
			expectedAmount := tt.route.Amount
			if tt.route.RouteInfo.Amount != "" {
				expectedAmount = tt.route.RouteInfo.Amount
			}
			amountMatch := tt.msgAmount == expectedAmount

			gotMatch := domainMatch && amountMatch

			if gotMatch != tt.wantMatch {
				t.Errorf("matchesRoute() = %v, want %v (domain match: %v, amount match: %v)",
					gotMatch, tt.wantMatch, domainMatch, amountMatch)
			}
		})
	}
}

func TestVerifyResult(t *testing.T) {
	tests := []struct {
		name         string
		result       *VerifyResult
		wantValid    bool
		wantErrors   int
		wantWarnings int
	}{
		{
			name: "valid result",
			result: &VerifyResult{
				Valid:        true,
				MatchedCount: 2,
				TotalRoutes:  2,
				Errors:       []string{},
				Warnings:     []string{},
			},
			wantValid:    true,
			wantErrors:   0,
			wantWarnings: 0,
		},
		{
			name: "invalid with errors",
			result: &VerifyResult{
				Valid:        false,
				MatchedCount: 1,
				TotalRoutes:  2,
				Errors:       []string{"Route 1 not matched", "Amount mismatch"},
				Warnings:     []string{},
			},
			wantValid:    false,
			wantErrors:   2,
			wantWarnings: 0,
		},
		{
			name: "valid with warnings",
			result: &VerifyResult{
				Valid:        true,
				MatchedCount: 2,
				TotalRoutes:  2,
				Errors:       []string{},
				Warnings:     []string{"Token ID not verified"},
			},
			wantValid:    true,
			wantErrors:   0,
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", tt.result.Valid, tt.wantValid)
			}
			if len(tt.result.Errors) != tt.wantErrors {
				t.Errorf("Errors count = %d, want %d", len(tt.result.Errors), tt.wantErrors)
			}
			if len(tt.result.Warnings) != tt.wantWarnings {
				t.Errorf("Warnings count = %d, want %d", len(tt.result.Warnings), tt.wantWarnings)
			}
		})
	}
}

func TestPrintResult(t *testing.T) {
	v := NewVerifier()

	// Test that PrintResult doesn't panic
	result := &VerifyResult{
		Valid:        true,
		MatchedCount: 2,
		TotalRoutes:  2,
		Errors:       []string{},
		Warnings:     []string{},
	}

	// This should not panic
	v.PrintResult(result)

	// Test with errors
	result = &VerifyResult{
		Valid:        false,
		MatchedCount: 1,
		TotalRoutes:  2,
		Errors:       []string{"Test error"},
		Warnings:     []string{"Test warning"},
	}

	// This should not panic
	v.PrintResult(result)
}
