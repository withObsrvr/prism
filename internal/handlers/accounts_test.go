package handlers

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

func TestBuildAccountDataRejectsInvalidAccountIDAsFormatError(t *testing.T) {
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest("GET", "/account/not-an-account", nil)

	_, err := h.buildAccountData(req, "mainnet", "not-an-account")
	if err == nil {
		t.Fatal("buildAccountData error = nil, want invalid account id format")
	}
	if !strings.Contains(err.Error(), "invalid account id format") {
		t.Fatalf("buildAccountData error = %q, want format wording", err)
	}
}

func TestApplySmartAccountStateToDataMapsRulesSignersAndPolicies(t *testing.T) {
	signerID := int64(10)
	policyID := int64(2)
	lastModified := int64(12345)
	data := pages.SmartAccountData{
		ContractID:           "CCONTRACT",
		BadgeLabel:           "Wallet-like Contract",
		ClassificationSource: "Observed",
		Confidence:           "low confidence",
	}
	state := &gateway.SmartAccountStateResponse{
		ContractID: "CCONTRACT",
		Source:     "silver.smart_account_state",
		Summary: gateway.SmartAccountContractSummary{
			ContractID:         "CCONTRACT",
			ContextRuleCount:   1,
			ActiveSignerCount:  1,
			ActivePolicyCount:  1,
			LastModifiedLedger: &lastModified,
		},
		Count: 1,
		ContextRules: []gateway.SmartAccountContextRuleRow{
			{
				ContextRuleID:      7,
				Active:             true,
				LastModifiedLedger: lastModified,
				TransactionHash:    "abcdef123456",
				Signers: []gateway.SmartAccountSignerRow{
					{
						SignerID:           &signerID,
						SignerType:         "external",
						CredentialID:       "9ca5204617ab254b6b21cbae8a30c42377d0cd4f",
						LastModifiedLedger: lastModified,
						RegistryResolved:   true,
					},
				},
				Policies: []gateway.SmartAccountPolicyRow{
					{
						PolicyID:           &policyID,
						PolicyAddress:      "CPOLICY",
						LastModifiedLedger: lastModified,
						RegistryResolved:   true,
					},
				},
			},
		},
	}

	applySmartAccountStateToData(&data, state)

	if data.BadgeLabel != "Smart Account" {
		t.Fatalf("BadgeLabel = %q, want Smart Account", data.BadgeLabel)
	}
	if data.ClassificationSource != "Event Index" || data.Confidence != "high confidence" {
		t.Fatalf("classification/confidence = %q/%q, want Event Index/high confidence", data.ClassificationSource, data.Confidence)
	}
	if len(data.Signers) != 1 {
		t.Fatalf("signers = %d, want 1", len(data.Signers))
	}
	if data.Signers[0].Role != "Rule 7" || data.Signers[0].KeyType != "EXTERNAL" {
		t.Fatalf("signer mapped unexpectedly: %+v", data.Signers[0])
	}
	if len(data.Policies) != 1 || len(data.Policies[0].Contracts) != 1 {
		t.Fatalf("policies mapped unexpectedly: %+v", data.Policies)
	}
	if len(data.SecurityLog) != 1 || data.SecurityLog[0].Action != "Rule 7 active" {
		t.Fatalf("security log mapped unexpectedly: %+v", data.SecurityLog)
	}
	if !containsString(data.Evidence, "last_modified_ledger:12345") {
		t.Fatalf("evidence missing last_modified_ledger: %+v", data.Evidence)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
