package handlers

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/intent"
)

func TestTransactionFailureRuleNamesStructuredOutcomeEvidence(t *testing.T) {
	got := ruleText(intent.Match{ID: intent.TransactionFailure})
	if !strings.Contains(got, "structured transaction outcome evidence") {
		t.Fatalf("ruleText = %q, want structured transaction outcome evidence", got)
	}
	if strings.Contains(got, "consolidated receipt") {
		t.Fatalf("ruleText retains receipt-only disclosure: %q", got)
	}
}
