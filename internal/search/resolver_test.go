package search

import (
	"strings"
	"testing"
)

func TestExtractIdentifierFromProse(t *testing.T) {
	tx := strings.Repeat("a", 64)
	account := "G" + strings.Repeat("A", 55)
	contract := "C" + strings.Repeat("B", 55)
	tests := []struct {
		input string
		type_ ClassType
		value string
	}{
		{"why did " + tx + " fail?", ClassTxHash, tx},
		{"open account " + account + " please", ClassAccount, account},
		{"show me contract " + contract, ClassContract, contract},
		{"what happened in ledger #12345?", ClassLedger, "12345"},
	}
	for _, test := range tests {
		got := ExtractIdentifier(test.input)
		if got.Type != test.type_ || got.Value != test.value {
			t.Errorf("ExtractIdentifier(%q) = %+v, want %s %q", test.input, got, test.type_, test.value)
		}
	}
}

func TestResolverRoutesSupportedQueriesExplicitly(t *testing.T) {
	resolver := NewResolver(nil)
	tx := strings.Repeat("a", 64)
	answerMatcher := func(input string) (AnswerCandidate, bool) {
		if strings.Contains(input, tx) && strings.Contains(strings.ToLower(input), "fail") {
			return AnswerCandidate{RuleID: "transaction_failure", Label: "Answer transaction failure", Summary: "Inspect decoded transaction failure evidence.", Confidence: 0.96, Slots: map[string]string{"tx_hash": tx}}, true
		}
		return AnswerCandidate{}, false
	}
	tests := []struct {
		query string
		kind  SearchResolutionKind
		rule  string
	}{
		{tx, SearchOpen, "entity.transaction"},
		{"why did " + tx + " fail?", SearchAnswer, "intent.transaction_failure"},
		{"failed USDC transfers last hour", SearchExplore, "activity.structured"},
		{"USDC transfers this week", SearchExplore, "activity.structured"},
		{"USDC", SearchOpen, "registry.asset"},
		{"explain whether the network feels healthy", SearchUnsupported, "unsupported.no_supported_rule"},
	}
	for _, test := range tests {
		got := resolver.Resolve(test.query, ResolveOptions{MatchAnswer: answerMatcher})
		if got.Kind != test.kind || got.RuleID != test.rule || got.Destination == "" || got.ActionLabel == "" {
			t.Errorf("Resolve(%q) = %+v, want kind=%s rule=%s", test.query, got, test.kind, test.rule)
		}
	}
}

func TestResolverDoesNotTurnUnsupportedProseIntoFreeTextExplore(t *testing.T) {
	for _, query := range []string{"tell me if the ecosystem looks unusual", "How active is USDC today?"} {
		got := NewResolver(nil).Resolve(query, ResolveOptions{})
		if got.Kind != SearchUnsupported || strings.Contains(got.Destination, "/v2/explore") {
			t.Fatalf("unsupported prose %q resolved as %+v", query, got)
		}
		if query == "How active is USDC today?" && got.RuleID != "unsupported.no_answer_rule" {
			t.Fatalf("ambiguous question rule = %q", got.RuleID)
		}
	}
}
