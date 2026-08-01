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
	// The hero used to carry a prose breakdown restating the envelope. That is
	// now the subtitle's job, in plain language and without the actor repeated.
	sub := operationsSubtitle(factsFromReceipt(data))
	for _, want := range []string{"placed a buy offer (6)", "placed a sell offer (7)"} {
		if !strings.Contains(sub, want) {
			t.Fatalf("subtitle missing %q: %s", want, sub)
		}
	}
	if strings.Contains(sub, "GABC") {
		t.Fatalf("subtitle repeats the actor already in the headline: %s", sub)
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

// The operations hero leads with the actor and what they did. It used to
// restate the operation count, which already appears in the chips, the sidebar,
// and the tab badge.
func TestOperationsHeroLeadsWithActorAndEconomicFact(t *testing.T) {
	data := legacy.TxReceiptData{
		SourceAddr: "GAPP...NEDY",
		OpsCount:   "3",
		Operations: []legacy.TxOperation{
			{Index: "1", Type: "Set Trust Flags", TypeName: "set_trust_line_flags"},
			{Index: "2", Type: "Buy Offer", TypeName: "manage_buy_offer", Amount: "111", Asset: "XLM"},
			{Index: "3", Type: "Set Trust Flags", TypeName: "set_trust_line_flags"},
		},
	}

	got := operationsTitleHTML(factsFromReceipt(data))
	for _, want := range []string{"GAPP...NEDY", "offered to buy", "111 XLM"} {
		if !strings.Contains(got, want) {
			t.Errorf("title missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "submitted 3 operations") {
		t.Errorf("title still restates the operation count: %s", got)
	}
	// The headline stays one clause; the other operations belong in the subtitle.
	if strings.Contains(got, "more operations") {
		t.Errorf("headline should not trail a count clause: %s", got)
	}

	sub := operationsSubtitle(factsFromReceipt(data))
	if sub != "Also set trust flags (2)." {
		t.Errorf("subtitle = %q, want %q", sub, "Also set trust flags (2).")
	}
	if strings.Contains(sub, "manage_buy_offer") || strings.Contains(sub, "set_trust_line_flags") {
		t.Errorf("subtitle leaks raw protocol names: %s", sub)
	}
}

// A manage_buy_offer places an offer; it does not necessarily execute. Saying
// "bought" would assert an economic event that may never have happened.
func TestOperationPhraseDoesNotOverstateOffers(t *testing.T) {
	tests := []struct {
		name    string
		op      legacy.TxOperation
		want    string
		mustNot []string
	}{
		{
			name:    "buy offer is an offer, not a purchase",
			op:      legacy.TxOperation{TypeName: "manage_buy_offer", Amount: "111", Asset: "XLM"},
			want:    "offered to buy 111 XLM",
			mustNot: []string{"bought", "purchased"},
		},
		{
			name:    "sell offer is an offer, not a sale",
			op:      legacy.TxOperation{TypeName: "manage_sell_offer", Amount: "40", Asset: "USDC"},
			want:    "offered to sell 40 USDC",
			mustNot: []string{"sold"},
		},
		{
			name: "payment did move value",
			op:   legacy.TxOperation{TypeName: "payment", Amount: "25", Asset: "USDC"},
			want: "sent 25 USDC",
		},
		{
			name: "valueless operation still phrases",
			op:   legacy.TxOperation{TypeName: "set_trust_line_flags"},
			want: "set trust flags",
		},
		{
			name: "api-prefixed spelling resolves",
			op:   legacy.TxOperation{TypeName: "OperationTypeManageBuyOffer", Amount: "5", Asset: "XLM"},
			want: "offered to buy 5 XLM",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := operationPhrase(tt.op)
			if got != tt.want {
				t.Errorf("operationPhrase() = %q, want %q", got, tt.want)
			}
			for _, bad := range tt.mustNot {
				if strings.Contains(got, bad) {
					t.Errorf("phrase %q overstates with %q", got, bad)
				}
			}
		})
	}
}

// Unknown operation types must not produce a bare actor with no verb.
func TestOperationsHeroFallsBackWhenNothingIsPhraseable(t *testing.T) {
	data := legacy.TxReceiptData{
		SourceAddr: "GAPP...NEDY",
		OpsCount:   "2",
		Operations: []legacy.TxOperation{
			{TypeName: "some_future_operation"},
			{TypeName: "another_future_operation"},
		},
	}
	got := operationsTitleHTML(factsFromReceipt(data))
	if !strings.Contains(got, "submitted") || !strings.Contains(got, "GAPP...NEDY") {
		t.Errorf("expected count fallback with actor, got %s", got)
	}
}

// Sampling 52 receipts across mainnet and testnet showed every hero variant
// putting the actor in the subtitle and a bare verb in the headline, so the
// headline was a subset of the line beneath it. Every variant now leads with
// the actor.
func TestEveryHeroVariantLeadsWithTheActor(t *testing.T) {
	const actor = "GACT...OR01"
	tests := []struct {
		name string
		data legacy.TxReceiptData
		want TxHeroKind
	}{
		{
			name: "generic call",
			data: legacy.TxReceiptData{SourceAddr: actor, IsSoroban: true, ContractFn: "fill_orders()", ContractAddr: "CAZV...BTZZ"},
			want: TxHeroGenericCall,
		},
		{
			name: "operations",
			data: legacy.TxReceiptData{SourceAddr: actor, SemanticTxType: "multi_op", OpsCount: "2", Operations: []legacy.TxOperation{
				{TypeName: "manage_buy_offer", Amount: "5", Asset: "XLM"}, {TypeName: "set_trust_line_flags"},
			}},
			want: TxHeroOperations,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hero := BuildTxHero(tt.data)
			if !strings.Contains(hero.TitleHTML, actor) {
				t.Errorf("%s headline does not lead with the actor: %s", tt.want, hero.TitleHTML)
			}
			if strings.Contains(hero.SubtitleHTML, "submitted a") && strings.Contains(hero.SubtitleHTML, "involving") {
				t.Errorf("%s subtitle restates the headline: %s", tt.want, hero.SubtitleHTML)
			}
		})
	}
}

// Setting an offer amount to zero is how Stellar cancels it. "offered to sell
// 0 XLM" appeared on real mainnet receipts before this.
func TestZeroAmountOfferReadsAsCancellation(t *testing.T) {
	for _, tt := range []struct{ typeName, want string }{
		{"manage_sell_offer", "cancelled a sell offer"},
		{"manage_buy_offer", "cancelled a buy offer"},
	} {
		got := operationPhrase(legacy.TxOperation{TypeName: tt.typeName, Amount: "0", Asset: "XLM"})
		if got != tt.want {
			t.Errorf("operationPhrase(%s, 0) = %q, want %q", tt.typeName, got, tt.want)
		}
	}
	// A zero-amount operation must not be chosen as the headline's lead when a
	// real value-bearing operation is present.
	ops := []legacy.TxOperation{
		{TypeName: "manage_sell_offer", Amount: "0", Asset: "XLM"},
		{TypeName: "manage_buy_offer", Amount: "13980.65", Asset: "XLM"},
	}
	if got := leadOperationIndex(ops); got != 1 {
		t.Errorf("leadOperationIndex = %d, want 1 (the operation carrying value)", got)
	}
}

// Grouping on the full phrase keyed every distinct amount separately, producing
// "offered to buy 13,980.65 XLM, offered to buy 27,961.3 XLM, ...".
func TestSubtitleGroupsRepeatedOperationsByKind(t *testing.T) {
	data := legacy.TxReceiptData{
		SourceAddr: "GAGW...ZUPL",
		Operations: []legacy.TxOperation{
			{TypeName: "manage_buy_offer", Amount: "13980.65", Asset: "XLM"},
			{TypeName: "manage_buy_offer", Amount: "27961.30", Asset: "XLM"},
			{TypeName: "manage_buy_offer", Amount: "41941.95", Asset: "XLM"},
			{TypeName: "manage_sell_offer", Amount: "14019.85", Asset: "XLM"},
		},
	}
	sub := operationsSubtitle(factsFromReceipt(data))
	if !strings.Contains(sub, "placed a buy offer (2)") || !strings.Contains(sub, "placed a sell offer") {
		t.Errorf("subtitle did not group by kind: %s", sub)
	}
	if strings.Contains(sub, "27,961") || strings.Contains(sub, "27961") {
		t.Errorf("subtitle repeats per-amount entries instead of grouping: %s", sub)
	}
}

// The gateway's placeholder value-flow phrase mis-pluralises at one.
func TestGenericTransferPhraseIsPluralisedLocally(t *testing.T) {
	for _, tt := range []struct {
		phrase string
		n      int
		ok     bool
	}{
		{"Transaction with 1 transfers", 1, true},
		{"Transaction with 8 transfers", 8, true},
		{"Swapped 5 XLM for 3 USDC", 0, false},
	} {
		n, ok := genericTransferCount(tt.phrase)
		if n != tt.n || ok != tt.ok {
			t.Errorf("genericTransferCount(%q) = (%d,%v), want (%d,%v)", tt.phrase, n, ok, tt.n, tt.ok)
		}
	}

	f := factsFromReceipt(legacy.TxReceiptData{SourceAddr: "GAIE...GRNG", HeroTitle: "Transaction with 1 transfers"})
	got := valueFlowTitleHTML(f)
	if !strings.Contains(got, "1 transfer<") || strings.Contains(got, "1 transfers") {
		t.Errorf("singular transfer mis-pluralised: %s", got)
	}
}

// Without structured outcome evidence Prism does not know what was attempted,
// so the headline must not narrate an action it cannot support.
func TestFailureHeadlineStaysHonestWithoutEvidence(t *testing.T) {
	f := factsFromReceipt(legacy.TxReceiptData{Status: "failed", SourceAddr: "GABC...WXYZ"})
	got := titleHTML(TxHeroFailure, f)
	if !strings.Contains(got, "Failure reason unavailable") {
		t.Errorf("headline invented a failure narrative: %s", got)
	}
}

// Lifecycle used to be inferred by substring: any operation whose display name
// contained "create", "merge", "extend" or "restore" reclassified the whole
// transaction. "Create Account" and "Create Claimable Balance" both matched, so
// a sponsored account-creation batch tagged "transfer" rendered the lifecycle
// hero, which has one entity slot and no counterparty, dropping the destination.
func TestLifecycleRequiresAnExactOperationMatch(t *testing.T) {
	tests := []struct {
		name          string
		op            legacy.TxOperation
		wantLifecycle bool
	}{
		{"create account moves value", legacy.TxOperation{TypeName: "create_account", Type: "Create Account"}, false},
		{"claimable balance moves value", legacy.TxOperation{TypeName: "create_claimable_balance", Type: "Create Claimable Balance"}, false},
		{"passive offer is trading", legacy.TxOperation{TypeName: "create_passive_sell_offer", Type: "Passive Offer"}, false},
		{"account merge destroys an account", legacy.TxOperation{TypeName: "account_merge", Type: "Account Merge"}, true},
		{"ttl extension", legacy.TxOperation{TypeName: "extend_footprint_ttl", Type: "Extend TTL"}, true},
		{"footprint restore", legacy.TxOperation{TypeName: "restore_footprint", Type: "Restore Footprint"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := factsFromReceipt(legacy.TxReceiptData{Status: "success", Operations: []legacy.TxOperation{tt.op}})
			if f.HasLifecycle != tt.wantLifecycle {
				t.Errorf("HasLifecycle = %v, want %v", f.HasLifecycle, tt.wantLifecycle)
			}
		})
	}
}

// The reported case: 19 sponsored account creations tagged "transfer". It must
// reach the value-flow hero, the only one with a counterparty slot.
func TestSponsoredAccountCreationBatchKeepsItsCounterparty(t *testing.T) {
	ops := make([]legacy.TxOperation, 0, 57)
	for i := 0; i < 19; i++ {
		ops = append(ops,
			legacy.TxOperation{TypeName: "begin_sponsoring_future_reserves", Type: "Begin Sponsorship"},
			legacy.TxOperation{TypeName: "create_account", Type: "Create Account", Amount: "10000", Asset: "XLM"},
			legacy.TxOperation{TypeName: "end_sponsoring_future_reserves", Type: "End Sponsorship"})
	}
	data := legacy.TxReceiptData{
		Status: "success", SemanticTxType: "transfer",
		SourceAddr: "GA22...MAOY", DestAddr: "GCPA...E6EL", Operations: ops,
	}
	hero := BuildTxHero(data)
	if hero.Kind != TxHeroValueFlow {
		t.Fatalf("kind = %q, want %q (lifecycle has no counterparty slot)", hero.Kind, TxHeroValueFlow)
	}
	if hero.ValueFlow == nil || hero.ValueFlow.ToAddress != "GCPA...E6EL" {
		t.Errorf("destination not carried into the hero: %+v", hero.ValueFlow)
	}
}

// Ranking the semantic classification above the lifecycle flag was tried and
// reverted; it turned account_merge into a plain transfer.
func TestAccountMergeStaysLifecycleEvenWhenTaggedTransfer(t *testing.T) {
	data := legacy.TxReceiptData{
		Status: "success", SemanticTxType: "transfer", SourceAddr: "GCX2...KXE6",
		Operations: []legacy.TxOperation{{TypeName: "account_merge", Type: "Account Merge"}},
	}
	hero := BuildTxHero(data)
	if hero.Kind != TxHeroLifecycle {
		t.Fatalf("kind = %q, want %q", hero.Kind, TxHeroLifecycle)
	}
	if !strings.Contains(hero.TitleHTML, "merged") {
		t.Errorf("headline lost the merge: %s", hero.TitleHTML)
	}
}

// The submitter signs the envelope; the event says whose balance changed. On a
// sponsored or relayed transfer these differ, and reading the sender from
// data.SourceAddr made the headline contradict the events tab.
func TestValueFlowSenderComesFromTheEventNotTheSigner(t *testing.T) {
	data := legacy.TxReceiptData{
		Status: "success", SemanticTxType: "transfer",
		SourceAddr: "GA22...MAOY", SourceAddrFull: "GA22FULL", DestAddr: "GWRONG...DEST",
		Events: []legacy.TxEvent{{
			Type: "transfer",
			From: "GCS7...ASOB", FromFull: "GCS7FULL",
			To: "GCPA...E6EL", ToFull: "GCPAFULL",
			Amount: "0", Asset: "token (CDLZ...CYSC)",
		}},
	}
	hero := BuildTxHero(data)
	if hero.Kind != TxHeroValueFlow {
		t.Fatalf("kind = %q, want %q", hero.Kind, TxHeroValueFlow)
	}
	if !strings.Contains(hero.TitleHTML, "GCS7...ASOB") {
		t.Errorf("headline does not name the event's sender: %s", hero.TitleHTML)
	}
	if strings.Contains(hero.TitleHTML, "GA22...MAOY") {
		t.Errorf("headline still names the signer as the sender: %s", hero.TitleHTML)
	}
	if !strings.Contains(hero.TitleHTML, "GCPA...E6EL") {
		t.Errorf("headline lost the destination: %s", hero.TitleHTML)
	}
	if hero.ValueFlow.FromAddress != "GCS7...ASOB" || hero.ValueFlow.ToAddress != "GCPA...E6EL" {
		t.Errorf("value flow parties not taken from the event: %+v", hero.ValueFlow)
	}
}

// With no event to read, the signer is the best available answer.
func TestValueFlowFallsBackToTheSignerWithoutEvents(t *testing.T) {
	data := legacy.TxReceiptData{
		Status: "success", SemanticTxType: "transfer",
		SourceAddr: "GA22...MAOY", DestAddr: "GCPA...E6EL",
		HeroTitle: "Transaction with 2 transfers",
	}
	hero := BuildTxHero(data)
	if !strings.Contains(hero.TitleHTML, "GA22...MAOY") {
		t.Errorf("fallback lost the actor: %s", hero.TitleHTML)
	}
	if hero.ValueFlow.FromAddress != "GA22...MAOY" {
		t.Errorf("fallback sender = %q, want the signer", hero.ValueFlow.FromAddress)
	}
}

// An event with no participants must not be chosen over one that has them.
func TestPrimaryTransferEventPrefersAValueBearingEvent(t *testing.T) {
	f := factsFromReceipt(legacy.TxReceiptData{Events: []legacy.TxEvent{
		{Type: "transfer", From: "", To: ""},
		{Type: "transfer", From: "GAAA...AAAA", To: "GBBB...BBBB"},
		{Type: "transfer", From: "GCCC...CCCC", To: "GDDD...DDDD", Amount: "5", Asset: "XLM"},
	}})
	e := primaryTransferEvent(f)
	if e == nil || e.From != "GCCC...CCCC" {
		t.Errorf("primaryTransferEvent = %+v, want the value-bearing event", e)
	}
}

func TestTransferVerbDistinguishesMintAndBurn(t *testing.T) {
	for in, want := range map[string]string{"transfer": "sent", "mint": "minted", "burn": "burned", "": "sent"} {
		if got := transferVerb(in); got != want {
			t.Errorf("transferVerb(%q) = %q, want %q", in, got, want)
		}
	}
}

// A sub-call that reverted inside a transaction that succeeded is the most
// notable thing about it, and no other surface on the receipt says so.
func TestRecoveredSubCallsSurfaceInTheSubtitle(t *testing.T) {
	tree := []legacy.TxCallNode{
		{Depth: 0, From: "GAPP...NEDY", To: "CBGS...KKY3", Function: "harvest", State: "ok"},
		{Depth: 1, From: "CBGS...KKY3", To: "CDL7...IGWA", Function: "harvest", State: "ok"},
		{Depth: 2, From: "CDL7...IGWA", To: "CB23...OUOV", Function: "mint", State: "ok"},
		{Depth: 1, From: "CBGS...KKY3", To: "CDL7...IGWA", Function: "harvest", State: "ok"},
		{Depth: 2, From: "CDL7...IGWA", To: "CB23...OUOV", Function: "mint", State: "ok"},
		{Depth: 1, From: "CBGS...KKY3", To: "CDL7...IGWA", Function: "harvest", State: "caught"},
	}
	hero := BuildTxHero(legacy.TxReceiptData{
		Status: "success", IsSoroban: true, SourceAddr: "GAPP...NEDY",
		ContractFn: "harvest()", ContractAddr: "CBGS...KKY3", CallTree: tree,
	})
	if hero.SubtitleHTML != "1 of 6 sub-calls reverted and was recovered." {
		t.Errorf("subtitle = %q", hero.SubtitleHTML)
	}
	// The transaction succeeded; a caught sub-call must not change that.
	if hero.Status != "Successful" {
		t.Errorf("status = %q, want Successful", hero.Status)
	}
}

func TestNoRecoveredNoteWhenEveryCallSucceeded(t *testing.T) {
	f := factsFromReceipt(legacy.TxReceiptData{CallTree: []legacy.TxCallNode{
		{State: "ok"}, {State: "ok"},
	}})
	if got := recoveredCallNote(f); got != "" {
		t.Errorf("note = %q, want empty", got)
	}
}

// Plural agreement: one reverted call reads differently from several.
func TestRecoveredCallNotePluralises(t *testing.T) {
	f := factsFromReceipt(legacy.TxReceiptData{CallTree: []legacy.TxCallNode{
		{State: "caught"}, {State: "caught"}, {State: "ok"},
	}})
	if got := recoveredCallNote(f); got != "2 of 3 sub-calls reverted and were recovered." {
		t.Errorf("note = %q", got)
	}
}
