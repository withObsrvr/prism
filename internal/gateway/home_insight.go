package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const (
	TTLHomeInsightReady      = 5 * time.Minute
	TTLHomeInsightImprovable = 15 * time.Second
	TTLHomeInsightFailure    = 10 * time.Second
)

var homeInsightIDPattern = regexp.MustCompile(`^hiev[12]_[A-Za-z0-9_-]{43}$`)

// HomeInsightDetailResponse is the retained evidence packet returned by the
// detail endpoint. The embedded preview fields intentionally preserve one
// interpretation path for the homepage and detail view.
type HomeInsightDetailResponse struct {
	HomeSummaryInsight
	Contributors []HomeInsightContribution `json:"contributors,omitempty"`
	Samples      []HomeInsightSample       `json:"samples,omitempty"`
}

type HomeInsightSample struct {
	SampleKind      string `json:"sample_kind"`
	Rank            int    `json:"rank"`
	LedgerSequence  int64  `json:"ledger_sequence"`
	TransactionHash string `json:"transaction_hash,omitempty"`
	OperationIndex  *int   `json:"operation_index,omitempty"`
	EventIndex      *int   `json:"event_index,omitempty"`
	ContractID      string `json:"contract_id,omitempty"`
	FunctionName    string `json:"function_name,omitempty"`
	ResultCode      string `json:"result_code,omitempty"`
	SelectionMethod string `json:"selection_method"`
}

type homeInsightNegativeEntry struct{ err error }

func ValidHomeInsightID(insightID string) bool {
	return homeInsightIDPattern.MatchString(insightID)
}

// GetHomeInsight returns one retained, bounded insight evidence packet.
func (c *Client) GetHomeInsight(ctx context.Context, network, insightID string) (*HomeInsightDetailResponse, error) {
	if !ValidHomeInsightID(insightID) {
		return nil, fmt.Errorf("gateway: invalid home insight ID")
	}
	cacheKey := fmt.Sprintf("%s:home_insight:%s", network, insightID)
	if v, ok := c.cache.Get(cacheKey); ok {
		if neg, isNeg := v.(*homeInsightNegativeEntry); isNeg {
			return nil, neg.err
		}
		return v.(*HomeInsightDetailResponse), nil
	}

	v, err, _ := c.inflight.DoContext(ctx, cacheKey, func() (any, error) {
		if v, ok := c.cache.Get(cacheKey); ok {
			if neg, isNeg := v.(*homeInsightNegativeEntry); isNeg {
				return nil, neg.err
			}
			return v, nil
		}

		body, err := c.doRequest(ctx, http.MethodGet, c.buildURL(network, "/home/insights/"+insightID))
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				c.cache.Set(cacheKey, &homeInsightNegativeEntry{err: err}, TTLHomeInsightFailure)
			}
			return nil, err
		}

		var packet HomeInsightDetailResponse
		if err := json.Unmarshal(body, &packet); err != nil {
			wrapped := fmt.Errorf("gateway: parsing home insight detail: %w", err)
			c.cache.Set(cacheKey, &homeInsightNegativeEntry{err: wrapped}, TTLHomeInsightFailure)
			return nil, wrapped
		}
		if packet.InsightID != insightID {
			return nil, fmt.Errorf("gateway: home insight ID mismatch")
		}
		if packet.Network != network {
			return nil, fmt.Errorf("gateway: home insight network mismatch")
		}

		ttl := TTLHomeInsightImprovable
		if packet.Status == "ready" {
			ttl = TTLHomeInsightReady
		}
		c.cache.Set(cacheKey, &packet, ttl)
		return &packet, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*HomeInsightDetailResponse), nil
}
