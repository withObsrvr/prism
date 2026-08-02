package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestBuildNormalizedTxContextPrefersInvokedContractOverContributingProtocolActor(t *testing.T) {
	const (
		invokedContract = "CDFWQCDY34ODL4IHTGOV6XSBKS3CRSCUYOITRKU5M3GQZBPO5UDH4364"
		tokenContract   = "CDA7SDCEQK2R6TTR655VNGEAONMNO3BSSRCFZDFNIJPADSMEKNEWRRBN"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"is_smart_wallet":false}`)
	}))
	defer server.Close()

	gw := gateway.New(
		gateway.Config{BaseURL: server.URL, APIKey: "test", Timeout: time.Second},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		context.Background(),
	)
	defer gw.Stop()
	h := &Handlers{Gateway: gw}
	tx := &gateway.TxFull{
		Operations: []gateway.DecodedOperation{{
			ContractID:   invokedContract,
			FunctionName: "create_and_try_fill_with_fee",
			IsSorobanOp:  true,
		}},
	}
	semantic := &gateway.SemanticTransactionResponse{
		Actors: []gateway.SemanticActor{
			// Token movement can cause a supporting SAC to appear before the
			// contract that received the top-level invocation. Actor ordering is
			// not evidence that the token was the transaction's primary target.
			{ActorID: tokenContract, ActorType: "contract", Roles: []string{"protocol"}},
			{ActorID: invokedContract, ActorType: "contract", Roles: []string{"protocol"}},
		},
	}

	got, err := h.buildNormalizedTxContext(context.Background(), "testnet", tx, semantic)
	if err != nil {
		t.Fatalf("buildNormalizedTxContext() error = %v", err)
	}
	if got.DownstreamContract == nil || got.DownstreamContract.ID != invokedContract {
		t.Fatalf("downstream contract = %+v, want invoked contract %s", got.DownstreamContract, invokedContract)
	}
	if got.DownstreamFunction != "create_and_try_fill_with_fee" {
		t.Fatalf("downstream function = %q", got.DownstreamFunction)
	}
}

func TestBuildNormalizedTxContextDoesNotInferFeePayerFromSource(t *testing.T) {
	h := &Handlers{}
	tx := &gateway.TxFull{Transaction: gateway.TxInfo{SourceAccount: "GAJDZBVLYPJOJRUQI2UL2C2BB52CHMKK2TWXPIFWDB4WGDKWD4K4DJBE"}}

	got, err := h.buildNormalizedTxContext(context.Background(), "testnet", tx, nil)
	if err != nil {
		t.Fatalf("buildNormalizedTxContext() error = %v", err)
	}
	if got.Submitter == nil || got.Submitter.ID != tx.Transaction.SourceAccount {
		t.Fatalf("submitter = %+v, want transaction source", got.Submitter)
	}
	if got.FeePayer != nil {
		t.Fatalf("fee payer was inferred without fee-bump evidence: %+v", got.FeePayer)
	}
}

func TestSelectSemanticEffectiveActorRejectsAmbiguousRoles(t *testing.T) {
	actors := []gateway.SemanticActor{
		{ActorID: "CCBD4XHNU2W6FTBZMEQSYPYUTXZNKMYV2HFWGPIWLSEP7GT5HAEI54S3", ActorType: "contract", Roles: []string{"effective_actor", "receiver"}},
		{ActorID: "CCBDP6MHOAASY2SJW3U3ZXYPV7YBQOJ2XOFVJMD3W5CKK2IOZ4I7L443", ActorType: "contract", Roles: []string{"effective_actor", "sender"}},
		{ActorID: "GAJDZBVLYPJOJRUQI2UL2C2BB52CHMKK2TWXPIFWDB4WGDKWD4K4DJBE", ActorType: "classic_account", Roles: []string{"effective_actor", "submitter"}},
	}

	if got := selectSemanticEffectiveActor(actors); got != nil {
		t.Fatalf("ambiguous effective actor was selected: %+v", got)
	}
}

func TestSelectSemanticEffectiveActorKeepsUniqueRole(t *testing.T) {
	actors := []gateway.SemanticActor{
		{ActorID: "CWALLET", ActorType: "smart_wallet", Roles: []string{"effective_actor"}},
		{ActorID: "CPROTOCOL", ActorType: "protocol", Roles: []string{"protocol"}},
	}

	got := selectSemanticEffectiveActor(actors)
	if got == nil || got.ActorID != "CWALLET" {
		t.Fatalf("unique effective actor = %+v, want CWALLET", got)
	}
}
