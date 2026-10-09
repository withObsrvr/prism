package jev

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAnalyzeQuerySendsRegistryAndDecodesAnalysis(t *testing.T) {
	t.Parallel()

	client := testClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		var request systemOneRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		state, ok := request.State.(map[string]any)
		if !ok || state["query"] != "Is Soroswap failing or just quiet?" {
			t.Fatalf("unexpected state: %#v", request.State)
		}
		if request.Model != "jev-test" || len(request.Questions) != 7 {
			t.Fatalf("unexpected model or registry: model=%q questions=%d", request.Model, len(request.Questions))
		}
		return jsonResponse(http.StatusOK, validResponse()), nil
	})

	analysis, err := client.AnalyzeQuery(context.Background(), "Is Soroswap failing or just quiet?")
	if err != nil {
		t.Fatal(err)
	}
	if analysis.RegistryVersion != AskRouterVersion || analysis.Model != "jev-1.13.0" {
		t.Fatalf("unexpected analysis identity: %#v", analysis)
	}
	if analysis.SubjectKind.Choice != "protocol" || analysis.RequestKind.Choice != "activity" {
		t.Fatalf("unexpected choices: %#v", analysis)
	}
	if analysis.AsksAboutFailures != 0.95 || analysis.AsksAboutActivity != 0.95 {
		t.Fatalf("unexpected facets: %#v", analysis)
	}
}

func TestAnalyzeQueryRejectsUnknownChoice(t *testing.T) {
	t.Parallel()

	client := testClient(t, func(*http.Request) (*http.Response, error) {
		body := strings.Replace(validResponse(), `"choice":"protocol"`, `"choice":"validator"`, 1)
		return jsonResponse(http.StatusOK, body), nil
	})
	_, err := client.AnalyzeQuery(context.Background(), "query")
	if err == nil || !strings.Contains(err.Error(), "unsupported choice") {
		t.Fatalf("expected unsupported choice error, got %v", err)
	}
}

func TestAnalyzeQueryRetriesRateLimit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := testClient(t, func(*http.Request) (*http.Response, error) {
		if calls.Add(1) < 3 {
			response := jsonResponse(http.StatusTooManyRequests, "")
			response.Header.Set("Retry-After", "0")
			return response, nil
		}
		return jsonResponse(http.StatusOK, validResponse()), nil
	})
	if _, err := client.AnalyzeQuery(context.Background(), "query"); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("expected 3 calls, got %d", got)
	}
}

func TestAnalyzeQueryHonorsCancellationDuringRetry(t *testing.T) {
	t.Parallel()

	client := testClient(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(529, ""), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.AnalyzeQuery(ctx, "query")
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestNewRequiresAPIKey(t *testing.T) {
	t.Parallel()
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected missing API key error")
	}
}

func testClient(t *testing.T, roundTrip roundTripFunc) *Client {
	t.Helper()
	client, err := New(Config{BaseURL: "https://example.test", APIKey: "test-key", Model: "jev-test", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTrip
	return client
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func validResponse() string {
	return `{
  "model":"jev-1.13.0",
  "answers":{
    "asks_about_failures":{"type":"noul","noul":0.95},
    "asks_about_activity":{"type":"noul","noul":0.95},
    "subject_kind":{"type":"choice","choice":"protocol","probabilities":{"transaction":0,"asset":0,"protocol":1,"contract":0,"network":0,"unknown":0},"confidence":1},
    "request_kind":{"type":"choice","choice":"activity","probabilities":{"activity":0.71,"failure_reason":0,"recent_failures":0.26,"expiration_risk":0,"general_information":0.03},"confidence":0.64},
    "evidence_scope":{"type":"choice","choice":"aggregate_pattern","probabilities":{"single_occurrence":0.16,"aggregate_pattern":0.63,"unclear":0.21},"confidence":0.45},
    "trend_kind":{"type":"choice","choice":"current_level","probabilities":{"current_level":0.41,"continuing":0.27,"recovery":0.1,"increase":0.08,"decrease":0.13,"none":0.01},"confidence":0.29},
    "time_window":{"type":"choice","choice":"current_snapshot","probabilities":{"one_hour":0,"one_day":0,"one_week":0,"one_month":0,"current_snapshot":0.87,"unspecified":0.13},"confidence":0.84}
  },
  "usage":{"input_tokens":500,"output_tokens":80}
}`
}
