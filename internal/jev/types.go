package jev

import (
	"context"
	"time"
)

const AskRouterVersion = "prism_ask_router_v1"

type Config struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

type Question struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type systemOneRequest struct {
	State     any                 `json:"state"`
	Model     string              `json:"model"`
	Questions map[string]Question `json:"questions"`
}

type systemOneResponse struct {
	Model   string               `json:"model"`
	Answers map[string]rawAnswer `json:"answers"`
	Usage   Usage                `json:"usage"`
}

type rawAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
	Noul          *float64           `json:"noul,omitempty"`
}

type ChoiceAnswer struct {
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type Analysis struct {
	RegistryVersion   string       `json:"registry_version"`
	Model             string       `json:"model"`
	SubjectKind       ChoiceAnswer `json:"subject_kind"`
	RequestKind       ChoiceAnswer `json:"request_kind"`
	EvidenceScope     ChoiceAnswer `json:"evidence_scope"`
	TrendKind         ChoiceAnswer `json:"trend_kind"`
	TimeWindow        ChoiceAnswer `json:"time_window"`
	AsksAboutFailures float64      `json:"asks_about_failures"`
	AsksAboutActivity float64      `json:"asks_about_activity"`
	Usage             Usage        `json:"usage"`
}

type Analyzer interface {
	AnalyzeQuery(ctx context.Context, query string) (Analysis, error)
}
