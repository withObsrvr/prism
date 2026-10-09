package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL     = "https://api.typesafe.ai"
	defaultModel       = "jev-latest"
	defaultTimeout     = 2 * time.Second
	maxResponseBytes   = 1 << 20
	maxAttempts        = 3
	defaultRetryDelay  = 100 * time.Millisecond
	maximumRetryDelay  = time.Second
	probabilityEpsilon = 0.02
)

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if !strings.HasPrefix(baseURL, "https://") && !strings.HasPrefix(baseURL, "http://") {
		return nil, errors.New("jev: base URL must use http or https")
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, errors.New("jev: API key is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) AnalyzeQuery(ctx context.Context, query string) (Analysis, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Analysis{}, errors.New("jev: query is required")
	}
	payload, err := json.Marshal(systemOneRequest{
		State:     map[string]string{"query": query},
		Model:     c.model,
		Questions: AskRouterQuestions(),
	})
	if err != nil {
		return Analysis{}, fmt.Errorf("jev: encode request: %w", err)
	}

	var response systemOneResponse
	for attempt := 0; attempt < maxAttempts; attempt++ {
		response, err = c.do(ctx, payload)
		var retryable *retryableError
		if !errors.As(err, &retryable) || attempt == maxAttempts-1 {
			break
		}
		if err := waitForRetry(ctx, retryable.delay, attempt); err != nil {
			return Analysis{}, err
		}
	}
	if err != nil {
		return Analysis{}, err
	}
	return buildAnalysis(response)
}

func (c *Client) do(ctx context.Context, payload []byte) (systemOneResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/systemone", bytes.NewReader(payload))
	if err != nil {
		return systemOneResponse{}, fmt.Errorf("jev: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return systemOneResponse{}, fmt.Errorf("jev: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 529 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return systemOneResponse{}, &retryableError{
			status: resp.StatusCode,
			delay:  retryAfter(resp.Header.Get("Retry-After")),
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return systemOneResponse{}, fmt.Errorf("jev: API returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return systemOneResponse{}, fmt.Errorf("jev: read response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return systemOneResponse{}, errors.New("jev: response exceeds size limit")
	}
	var decoded systemOneResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return systemOneResponse{}, fmt.Errorf("jev: decode response: %w", err)
	}
	return decoded, nil
}

type retryableError struct {
	status int
	delay  time.Duration
}

func (e *retryableError) Error() string {
	return fmt.Sprintf("jev: API returned retryable HTTP %d", e.status)
}

func retryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds < 0 {
		return 0
	}
	d := time.Duration(seconds) * time.Second
	if d > maximumRetryDelay {
		return maximumRetryDelay
	}
	return d
}

func waitForRetry(ctx context.Context, requested time.Duration, attempt int) error {
	delay := requested
	if delay <= 0 {
		delay = defaultRetryDelay * time.Duration(1<<attempt)
	}
	if delay > maximumRetryDelay {
		delay = maximumRetryDelay
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("jev: retry cancelled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func buildAnalysis(response systemOneResponse) (Analysis, error) {
	if strings.TrimSpace(response.Model) == "" {
		return Analysis{}, errors.New("jev: response model is missing")
	}
	if response.Usage.InputTokens < 0 || response.Usage.OutputTokens < 0 {
		return Analysis{}, errors.New("jev: response contains invalid token usage")
	}

	subject, err := requiredChoice(response.Answers, "subject_kind", optionSet("transaction", "asset", "protocol", "contract", "network", "unknown"))
	if err != nil {
		return Analysis{}, err
	}
	request, err := requiredChoice(response.Answers, "request_kind", optionSet("activity", "failure_reason", "recent_failures", "expiration_risk", "general_information"))
	if err != nil {
		return Analysis{}, err
	}
	scope, err := requiredChoice(response.Answers, "evidence_scope", optionSet("single_occurrence", "aggregate_pattern", "unclear"))
	if err != nil {
		return Analysis{}, err
	}
	trend, err := requiredChoice(response.Answers, "trend_kind", optionSet("current_level", "continuing", "recovery", "increase", "decrease", "none"))
	if err != nil {
		return Analysis{}, err
	}
	window, err := requiredChoice(response.Answers, "time_window", optionSet("one_hour", "one_day", "one_week", "one_month", "current_snapshot", "unspecified"))
	if err != nil {
		return Analysis{}, err
	}
	failures, err := requiredNoul(response.Answers, "asks_about_failures")
	if err != nil {
		return Analysis{}, err
	}
	activity, err := requiredNoul(response.Answers, "asks_about_activity")
	if err != nil {
		return Analysis{}, err
	}

	return Analysis{
		RegistryVersion:   AskRouterVersion,
		Model:             response.Model,
		SubjectKind:       subject,
		RequestKind:       request,
		EvidenceScope:     scope,
		TrendKind:         trend,
		TimeWindow:        window,
		AsksAboutFailures: failures,
		AsksAboutActivity: activity,
		Usage:             response.Usage,
	}, nil
}

func requiredChoice(answers map[string]rawAnswer, id string, allowed map[string]struct{}) (ChoiceAnswer, error) {
	answer, ok := answers[id]
	if !ok {
		return ChoiceAnswer{}, fmt.Errorf("jev: response is missing %s", id)
	}
	if answer.Type != "choice" || answer.Confidence == nil {
		return ChoiceAnswer{}, fmt.Errorf("jev: %s has the wrong answer type", id)
	}
	if _, ok := allowed[answer.Choice]; !ok {
		return ChoiceAnswer{}, fmt.Errorf("jev: %s returned unsupported choice %q", id, answer.Choice)
	}
	if !unitInterval(*answer.Confidence) {
		return ChoiceAnswer{}, fmt.Errorf("jev: %s returned invalid confidence", id)
	}
	if len(answer.Probabilities) != len(allowed) {
		return ChoiceAnswer{}, fmt.Errorf("jev: %s returned an incomplete probability distribution", id)
	}
	total := 0.0
	for option, probability := range answer.Probabilities {
		if _, ok := allowed[option]; !ok || !unitInterval(probability) {
			return ChoiceAnswer{}, fmt.Errorf("jev: %s returned an invalid probability distribution", id)
		}
		total += probability
	}
	if math.Abs(total-1) > probabilityEpsilon {
		return ChoiceAnswer{}, fmt.Errorf("jev: %s probabilities do not sum to one", id)
	}
	return ChoiceAnswer{
		Choice:        answer.Choice,
		Probabilities: answer.Probabilities,
		Confidence:    *answer.Confidence,
	}, nil
}

func requiredNoul(answers map[string]rawAnswer, id string) (float64, error) {
	answer, ok := answers[id]
	if !ok {
		return 0, fmt.Errorf("jev: response is missing %s", id)
	}
	if answer.Type != "noul" || answer.Noul == nil || !unitInterval(*answer.Noul) {
		return 0, fmt.Errorf("jev: %s returned an invalid noul answer", id)
	}
	return *answer.Noul, nil
}

func optionSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func unitInterval(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}
