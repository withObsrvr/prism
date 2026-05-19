package search

import (
	"net/url"
	"sort"
	"strings"
)

type Query struct {
	Scope  string
	Topic  string
	Fn     string
	Asset  string
	Time   string
	Status string
	Free   string
}

var topicWords = map[string]string{
	"swap": "swap", "swaps": "swap", "swapped": "swap", "trade": "swap", "trades": "swap",
	"transfer": "transfer", "transfers": "transfer", "payment": "transfer", "payments": "transfer", "send": "transfer", "sent": "transfer",
	"mint": "mint", "mints": "mint", "minted": "mint",
	"burn": "burn", "burns": "burn", "burned": "burn",
	"approve": "approve", "approves": "approve", "approval": "approve", "allowance": "approve",
	"deposit": "deposit", "deposits": "deposit", "supply": "deposit", "supplied": "deposit",
	"withdraw": "withdraw", "withdraws": "withdraw", "borrow": "withdraw", "borrowed": "withdraw",
}

var functionWords = map[string]string{
	"swap": "swap", "add_liquidity": "add_liquidity", "deposit": "deposit", "withdraw": "withdraw",
	"approve": "approve", "transfer": "transfer", "mint": "mint", "burn": "burn", "submit": "submit",
}

var assetWords = map[string]string{
	"XLM": "XLM", "USDC": "USDC", "EURC": "EURC", "BTC": "BTC", "ETH": "ETH",
	"AQUA": "AQUA", "BLND": "BLND", "PHO": "PHO", "SCALE": "SCALE",
}

func Parse(input string) (Query, float64) {
	q := Query{Scope: "all", Time: "1h"}
	text := strings.TrimSpace(input)
	if text == "" {
		return q, 0
	}

	normalized := strings.NewReplacer(",", " ", ".", " ", "?", " ", "!", " ").Replace(text)
	tokens := strings.Fields(normalized)
	consumed := make([]bool, len(tokens))
	matches := 0

	for i, raw := range tokens {
		low := strings.ToLower(raw)
		up := strings.ToUpper(raw)
		switch low {
		case "all", "every", "any":
			q.Scope = "all"
			consumed[i] = true
			matches++
		case "soroban", "contract", "contracts", "smart":
			q.Scope = "soroban"
			consumed[i] = true
			matches++
		case "classic", "stellar":
			q.Scope = "classic"
			consumed[i] = true
			matches++
		case "failed", "fail", "failure", "unsuccessful":
			q.Status = "failed"
			consumed[i] = true
			matches++
		case "successful", "success", "succeeded":
			q.Status = "success"
			consumed[i] = true
			matches++
		case "today", "day":
			q.Time = "24h"
			consumed[i] = true
			matches++
		case "hour", "hourly":
			q.Time = "1h"
			consumed[i] = true
			matches++
		case "week", "weekly":
			q.Time = "7d"
			consumed[i] = true
			matches++
		case "month", "monthly":
			q.Time = "30d"
			consumed[i] = true
			matches++
		case "last", "past", "recent", "recently", "me", "show", "find", "for", "in", "the", "this", "activity", "transactions", "txs", "operations", "ops", "calls", "calling", "events":
			consumed[i] = true
		}
		if topic, ok := topicWords[low]; ok {
			q.Topic = topic
			consumed[i] = true
			matches++
			if fn, ok := functionWords[low]; ok {
				q.Fn = fn
			} else if topic == "swap" || topic == "transfer" || topic == "mint" || topic == "burn" || topic == "approve" || topic == "deposit" || topic == "withdraw" {
				q.Fn = topic
			}
			continue
		}
		if fn, ok := functionWords[low]; ok {
			q.Fn = fn
			consumed[i] = true
			matches++
			continue
		}
		if asset, ok := assetWords[up]; ok {
			q.Asset = asset
			consumed[i] = true
			matches++
			continue
		}
		if len(up) >= 2 && len(up) <= 12 && up == raw && looksAssetCode(up) {
			q.Asset = up
			consumed[i] = true
			matches++
		}
	}

	var free []string
	for i, token := range tokens {
		if !consumed[i] {
			free = append(free, token)
		}
	}
	q.Free = strings.Join(free, " ")
	confidence := float64(matches) / float64(max(1, len(tokens)))
	if q.Topic != "" || q.Fn != "" || q.Asset != "" {
		confidence += 0.35
	}
	if q.Status != "" || q.Scope != "all" || q.Time != "1h" {
		confidence += 0.1
	}
	if confidence > 1 {
		confidence = 1
	}
	return q, confidence
}

func (q Query) QueryString() string {
	v := url.Values{}
	if q.Scope != "" && q.Scope != "all" {
		v.Set("scope", q.Scope)
	}
	if q.Topic != "" {
		v.Set("topic", q.Topic)
	}
	if q.Fn != "" {
		v.Set("fn", q.Fn)
	}
	if q.Asset != "" {
		v.Set("asset", q.Asset)
	}
	if q.Time != "" && q.Time != "1h" {
		v.Set("time", q.Time)
	}
	if q.Status != "" {
		v.Set("status", q.Status)
	}
	if q.Free != "" {
		v.Set("q", q.Free)
	}
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := url.Values{}
	for _, k := range keys {
		out[k] = v[k]
	}
	return out.Encode()
}

func looksAssetCode(s string) bool {
	for _, c := range s {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
