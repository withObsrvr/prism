package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestDetectSmartRedirectRoutesSmartWalletToV2SmartAccount(t *testing.T) {
	const contractID = "CCQBQIAG2E2L5NOIML2SGAJYMXPID3MAQNII5USMENID3SDJ4ATOU2HG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/smart-accounts/" + contractID + "/rules":
			http.NotFound(w, r)
		case "/lake/v1/testnet/api/v1/silver/smart-wallet/" + contractID:
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"contract_id":%q,"is_smart_wallet":true,"wallet_type":"openzeppelin"}`, contractID)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	gw := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer gw.Stop()
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Gateway: gw}

	got := h.detectSmartRedirect(context.Background(), "testnet", contractID)
	want := "/v2/account/" + contractID + "/smart"
	if got != want {
		t.Fatalf("detectSmartRedirect = %q, want %q", got, want)
	}
}

func TestDetectSmartRedirectRoutesSmartAccountRulesToV2SmartAccount(t *testing.T) {
	const contractID = "CCQBQIAG2E2L5NOIML2SGAJYMXPID3MAQNII5USMENID3SDJ4ATOU2HG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/smart-accounts/" + contractID + "/rules":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"contract_id":%q,"summary":{"contract_id":%q,"wallet_type":"openzeppelin"},"count":1}`, contractID, contractID)
		case "/lake/v1/testnet/api/v1/silver/smart-wallet/" + contractID:
			t.Fatalf("smart-wallet fallback should not be called when rules exist")
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	gw := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer gw.Stop()
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Gateway: gw}

	got := h.detectSmartRedirect(context.Background(), "testnet", contractID)
	want := "/v2/account/" + contractID + "/smart"
	if got != want {
		t.Fatalf("detectSmartRedirect = %q, want %q", got, want)
	}
}

func TestGAccountDetailV2RedirectsSmartAccountContractToSmartView(t *testing.T) {
	const contractID = "CCQBQIAG2E2L5NOIML2SGAJYMXPID3MAQNII5USMENID3SDJ4ATOU2HG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/smart-accounts/" + contractID + "/rules":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"contract_id":%q,"summary":{"contract_id":%q,"wallet_type":"openzeppelin"},"count":1}`, contractID, contractID)
		case "/lake/v1/testnet/api/v1/silver/smart-wallet/" + contractID:
			t.Fatalf("smart-wallet fallback should not be called when rules exist")
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	gw := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer gw.Stop()
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Gateway: gw}

	req := httptest.NewRequest(http.MethodGet, "/v2/account/"+contractID+"?network=testnet", nil)
	req.SetPathValue("id", contractID)
	rec := httptest.NewRecorder()

	h.GAccountDetailV2(rec, req)

	want := "/v2/account/" + contractID + "/smart"
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != want {
		t.Fatalf("redirect = %d %q, want %d %q", rec.Code, rec.Header().Get("Location"), http.StatusSeeOther, want)
	}
}

func TestBuildSmartAccountDataUsesRulesWithoutSlowEnrichment(t *testing.T) {
	const contractID = "CCQBQIAG2E2L5NOIML2SGAJYMXPID3MAQNII5USMENID3SDJ4ATOU2HG"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lake/v1/testnet/api/v1/silver/smart-accounts/" + contractID + "/rules":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{
				"contract_id":%q,
				"summary":{"contract_id":%q,"wallet_type":"openzeppelin","context_rule_count":1,"active_signer_count":1,"active_policy_count":1},
				"count":1,
				"context_rules":[{"context_rule_id":1,"active":true,"signers":[{"signer_type":"external","credential_id":"cred"}],"policies":[{"policy_address":"CPOLICY"}]}]
			}`, contractID, contractID)
		case "/lake/v1/testnet/api/v1/silver/smart-wallets/" + contractID,
			"/lake/v1/testnet/api/v1/silver/smart-wallet/" + contractID,
			"/lake/v1/testnet/api/v1/silver/transfers":
			t.Fatalf("slow endpoint should not be called on rules-backed smart account render: %s", r.URL.String())
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	gw := gateway.New(gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), context.Background())
	defer gw.Stop()
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Gateway: gw}
	req := httptest.NewRequest(http.MethodGet, "/v2/account/"+contractID+"/smart?network=testnet", nil)
	req.SetPathValue("id", contractID)

	data, err := h.buildSmartAccountData(req, "testnet", contractID)
	if err != nil {
		t.Fatalf("buildSmartAccountData error = %v", err)
	}
	if data.ContractID != contractID || data.ClassificationSource != "Event Index" {
		t.Fatalf("unexpected smart account data: %+v", data)
	}
	if len(data.Signers) != 1 || len(data.Policies) != 1 {
		t.Fatalf("rules data not mapped: signers=%d policies=%d", len(data.Signers), len(data.Policies))
	}
	if len(data.ActivityLog) != 0 {
		t.Fatalf("activity should not be loaded in primary render, got %d rows", len(data.ActivityLog))
	}
}

func TestDirectSuggestionLabelNamesSmartAccount(t *testing.T) {
	const contractID = "CCQBQIAG2E2L5NOIML2SGAJYMXPID3MAQNII5USMENID3SDJ4ATOU2HG"
	got := directSuggestionLabel(contractID, "/v2/account/"+contractID+"/smart")
	if !strings.HasPrefix(got, "Smart Account ") {
		t.Fatalf("directSuggestionLabel = %q, want Smart Account label", got)
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
