package jev

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONLRecorderAndLedgerEvaluationReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shadow.jsonl")
	recorder, err := NewJSONLRecorder(path)
	if err != nil {
		t.Fatal(err)
	}
	records := []LedgerShadowRecord{
		evaluationRecord(1, "succeeded", "widespread_transaction_failures", "widespread_transaction_failures", .9),
		evaluationRecord(2, "succeeded", "evidence_incomplete", "elevated_fees", .85),
		evaluationRecord(3, "error", "evidence_incomplete", "", 0),
	}
	records[0].Agrees = true
	records[0].PreferredLead = records[0].Deterministic
	records[1].PreferredLead = records[1].ShadowLead
	records[2].Reason = "timeout_or_cancellation"
	records = append(records, records[1])
	for _, record := range records {
		if err := recorder.RecordLedgerShadow(record); err != nil {
			t.Fatal(err)
		}
	}
	if err := recorder.Close(); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	report, err := EvaluateLedgerShadow(file)
	if err != nil {
		t.Fatal(err)
	}
	if report.InputRecords != 4 || report.DuplicateRecords != 1 || report.Records != 3 || report.Succeeded != 2 || report.Agreements != 1 || report.Disagreements != 1 || report.AgreementRate != .5 {
		t.Fatalf("report = %+v", report)
	}
	if report.Reviewed != 2 || report.HumanMatchesDeterministic != 1 || report.HumanMatchesShadow != 2 {
		t.Fatalf("human labels = %+v", report)
	}
	if len(report.ReviewQueue) != 2 || report.ReviewQueue[0].Reason != "high_confidence_disagreement" {
		t.Fatalf("review queue = %+v", report.ReviewQueue)
	}
}

func evaluationRecord(sequence int64, status, deterministic, shadow string, confidence float64) LedgerShadowRecord {
	return LedgerShadowRecord{
		RecordVersion: LedgerShadowRecordVersion, Network: "testnet", LedgerSequence: sequence,
		Bands: map[string]string{"evidence": "complete"}, Eligible: []string{deterministic, shadow},
		Deterministic: deterministic, ShadowStatus: status, ShadowLead: shadow,
		Probabilities: map[string]float64{deterministic: 1 - confidence, shadow: confidence},
		Confidence:    confidence, Model: "jev-test", RegistryVersion: LedgerSelectorRegistryVersion,
		BandVersion: "bands", BaselineVersion: "baselines", ThresholdVersion: "thresholds", TemplateVersion: "templates", SelectorVersion: "selector", ShadowIdentity: "shadow:test",
		LatencyMS: 25, InputTokens: 10, OutputTokens: 2,
	}
}
