package viewmodelv2

import (
	"strings"
	"testing"

	legacy "github.com/withObsrvr/prism/internal/templates/pages"
)

func TestBuildTxHeroClassifiesClassicMultiOperationEnvelope(t *testing.T) {
	data := classicOfferTransactionFixture()
	hero := BuildTxHero(data)

	if hero.Kind != TxHeroOperations {
		t.Fatalf("kind = %q, want %q", hero.Kind, TxHeroOperations)
	}
	if hero.Operations == nil {
		t.Fatal("operations hero is nil")
	}
	if hero.Operations.Count != 14 {
		t.Fatalf("operation count = %d, want 14", hero.Operations.Count)
	}
	if strings.Contains(strings.ToLower(hero.TitleHTML), "called") || strings.Contains(strings.ToLower(hero.TitleHTML), "contract") {
		t.Fatalf("classic operation title was labeled as a contract call: %s", hero.TitleHTML)
	}
	for _, want := range []string{"7</b> manage buy offer operations", "7</b> manage sell offer operations"} {
		if !strings.Contains(hero.Operations.SummaryHTML, want) {
			t.Fatalf("summary missing %q: %s", want, hero.Operations.SummaryHTML)
		}
	}
}

func TestBuildTxHeroKeepsSorobanMultiOperationEnvelopeOnContractPath(t *testing.T) {
	data := classicOfferTransactionFixture()
	data.IsSoroban = true
	data.Operations[0].IsSoroban = true
	data.Operations[0].Type = "Invoke Contract"
	data.Operations[0].Contract = "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSC4"
	data.Operations[0].Function = "swap()"

	hero := BuildTxHero(data)
	if hero.Kind == TxHeroOperations {
		t.Fatalf("Soroban envelope incorrectly classified as classic operations: %#v", hero)
	}
}

func classicOfferTransactionFixture() legacy.TxReceiptData {
	operations := make([]legacy.TxOperation, 0, 14)
	for i := 0; i < 7; i++ {
		operations = append(operations, legacy.TxOperation{Index: "buy", Type: "Manage Buy Offer", Status: "Success"})
		operations = append(operations, legacy.TxOperation{Index: "sell", Type: "Manage Sell Offer", Status: "Success"})
	}
	return legacy.TxReceiptData{
		Status:              "success",
		SemanticTxType:      "multi_op",
		EffectiveActorShort: "GABC...WXYZ",
		EffectiveActorAddr:  "GABC",
		OpsCount:            "14",
		Operations:          operations,
	}
}
