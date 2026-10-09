package jev

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const LedgerShadowRecordVersion = "prism_ledger_shadow_record_v1"

type LedgerShadowRecord struct {
	RecordVersion    string             `json:"record_version"`
	Network          string             `json:"network"`
	LedgerSequence   int64              `json:"ledger_sequence"`
	Bands            map[string]string  `json:"bands"`
	Eligible         []string           `json:"eligible"`
	Deterministic    string             `json:"deterministic_lead"`
	ShadowStatus     string             `json:"shadow_status"`
	ShadowLead       string             `json:"shadow_lead,omitempty"`
	Probabilities    map[string]float64 `json:"probabilities,omitempty"`
	Confidence       float64            `json:"confidence,omitempty"`
	Agrees           bool               `json:"agrees"`
	Reason           string             `json:"reason,omitempty"`
	Model            string             `json:"model,omitempty"`
	RegistryVersion  string             `json:"registry_version,omitempty"`
	BandVersion      string             `json:"band_version"`
	BaselineVersion  string             `json:"baseline_version"`
	ThresholdVersion string             `json:"threshold_version"`
	TemplateVersion  string             `json:"template_version"`
	SelectorVersion  string             `json:"selector_version"`
	ShadowIdentity   string             `json:"shadow_identity"`
	LatencyMS        int64              `json:"latency_ms"`
	InputTokens      int                `json:"input_tokens,omitempty"`
	OutputTokens     int                `json:"output_tokens,omitempty"`
	PreferredLead    string             `json:"preferred_lead,omitempty"`
	ReviewerNote     string             `json:"reviewer_note,omitempty"`
}

type LedgerShadowRecorder interface {
	RecordLedgerShadow(LedgerShadowRecord) error
}

type JSONLRecorder struct {
	mu   sync.Mutex
	file *os.File
}

func NewJSONLRecorder(path string) (*JSONLRecorder, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("jev: shadow log path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("jev: create shadow log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("jev: open shadow log: %w", err)
	}
	return &JSONLRecorder{file: file}, nil
}

func (r *JSONLRecorder) RecordLedgerShadow(record LedgerShadowRecord) error {
	if r == nil || r.file == nil {
		return errors.New("jev: shadow recorder is closed")
	}
	if record.RecordVersion != LedgerShadowRecordVersion || record.Network == "" || record.LedgerSequence <= 0 || record.Deterministic == "" || record.ShadowStatus == "" || record.ShadowIdentity == "" {
		return errors.New("jev: invalid ledger shadow record")
	}
	body, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("jev: encode shadow record: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.file.Write(append(body, '\n')); err != nil {
		return fmt.Errorf("jev: write shadow record: %w", err)
	}
	return nil
}

func (r *JSONLRecorder) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

type LedgerEvaluationReport struct {
	RecordVersion             string             `json:"record_version"`
	InputRecords              int                `json:"input_records"`
	Records                   int                `json:"records"`
	DuplicateRecords          int                `json:"duplicate_records"`
	Succeeded                 int                `json:"succeeded"`
	Agreements                int                `json:"agreements"`
	Disagreements             int                `json:"disagreements"`
	AgreementRate             float64            `json:"agreement_rate"`
	AverageConfidence         float64            `json:"average_confidence"`
	ConfidenceBands           map[string]int     `json:"confidence_bands"`
	Statuses                  map[string]int     `json:"statuses"`
	DeterministicLeads        map[string]int     `json:"deterministic_leads"`
	ShadowLeads               map[string]int     `json:"shadow_leads"`
	DisagreementPairs         map[string]int     `json:"disagreement_pairs"`
	AverageLatencyMS          float64            `json:"average_latency_ms"`
	InputTokens               int                `json:"input_tokens"`
	OutputTokens              int                `json:"output_tokens"`
	Reviewed                  int                `json:"reviewed"`
	HumanMatchesDeterministic int                `json:"human_matches_deterministic"`
	HumanMatchesShadow        int                `json:"human_matches_shadow"`
	ReviewQueue               []LedgerReviewCase `json:"review_queue"`
}

type LedgerReviewCase struct {
	Network        string  `json:"network"`
	LedgerSequence int64   `json:"ledger_sequence"`
	Reason         string  `json:"reason"`
	Deterministic  string  `json:"deterministic_lead"`
	ShadowLead     string  `json:"shadow_lead,omitempty"`
	Confidence     float64 `json:"confidence,omitempty"`
}

func EvaluateLedgerShadow(reader io.Reader) (LedgerEvaluationReport, error) {
	report := LedgerEvaluationReport{RecordVersion: LedgerShadowRecordVersion, ConfidenceBands: map[string]int{}, Statuses: map[string]int{}, DeterministicLeads: map[string]int{}, ShadowLeads: map[string]int{}, DisagreementPairs: map[string]int{}}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	var confidence, latency float64
	seen := map[string]struct{}{}
	for line := 1; scanner.Scan(); line++ {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var record LedgerShadowRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return LedgerEvaluationReport{}, fmt.Errorf("decode shadow record line %d: %w", line, err)
		}
		if record.RecordVersion != LedgerShadowRecordVersion {
			return LedgerEvaluationReport{}, fmt.Errorf("unsupported shadow record version %q on line %d", record.RecordVersion, line)
		}
		report.InputRecords++
		identity := fmt.Sprintf("%s:%d:%s:%s:%s:%s:%s", record.Network, record.LedgerSequence, record.ShadowIdentity, record.BandVersion, record.BaselineVersion, record.ThresholdVersion, record.SelectorVersion)
		if _, exists := seen[identity]; exists {
			report.DuplicateRecords++
			continue
		}
		seen[identity] = struct{}{}
		report.Records++
		report.Statuses[record.ShadowStatus]++
		report.DeterministicLeads[record.Deterministic]++
		latency += float64(record.LatencyMS)
		report.InputTokens += record.InputTokens
		report.OutputTokens += record.OutputTokens
		if record.PreferredLead != "" {
			report.Reviewed++
			if record.PreferredLead == record.Deterministic {
				report.HumanMatchesDeterministic++
			}
			if record.PreferredLead == record.ShadowLead {
				report.HumanMatchesShadow++
			}
		}
		if record.ShadowStatus != "succeeded" {
			report.ReviewQueue = append(report.ReviewQueue, reviewCase(record, record.ShadowStatus+":"+record.Reason))
			continue
		}
		report.Succeeded++
		report.ShadowLeads[record.ShadowLead]++
		confidence += record.Confidence
		switch {
		case record.Confidence < .5:
			report.ConfidenceBands["low_lt_0.50"]++
		case record.Confidence < .8:
			report.ConfidenceBands["medium_0.50_to_0.79"]++
		default:
			report.ConfidenceBands["high_gte_0.80"]++
		}
		if record.Agrees {
			report.Agreements++
		} else {
			report.Disagreements++
			report.DisagreementPairs[record.Deterministic+" -> "+record.ShadowLead]++
		}
		if !record.Agrees || record.Confidence < .5 {
			reason := "disagreement"
			if record.Confidence < .5 {
				reason = "low_confidence"
			}
			if !record.Agrees && record.Confidence >= .8 {
				reason = "high_confidence_disagreement"
			}
			report.ReviewQueue = append(report.ReviewQueue, reviewCase(record, reason))
		}
	}
	if err := scanner.Err(); err != nil {
		return LedgerEvaluationReport{}, fmt.Errorf("read shadow records: %w", err)
	}
	if report.Succeeded > 0 {
		report.AgreementRate = float64(report.Agreements) / float64(report.Succeeded)
		report.AverageConfidence = confidence / float64(report.Succeeded)
	}
	if report.Records > 0 {
		report.AverageLatencyMS = latency / float64(report.Records)
	}
	sort.Slice(report.ReviewQueue, func(i, j int) bool {
		if report.ReviewQueue[i].Confidence != report.ReviewQueue[j].Confidence {
			return report.ReviewQueue[i].Confidence > report.ReviewQueue[j].Confidence
		}
		if report.ReviewQueue[i].Network != report.ReviewQueue[j].Network {
			return report.ReviewQueue[i].Network < report.ReviewQueue[j].Network
		}
		return report.ReviewQueue[i].LedgerSequence < report.ReviewQueue[j].LedgerSequence
	})
	return report, nil
}

func reviewCase(record LedgerShadowRecord, reason string) LedgerReviewCase {
	return LedgerReviewCase{Network: record.Network, LedgerSequence: record.LedgerSequence, Reason: reason, Deterministic: record.Deterministic, ShadowLead: record.ShadowLead, Confidence: record.Confidence}
}
