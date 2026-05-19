package search

import (
	"sort"
	"strings"
)

type EntityType string

const (
	EntityContract EntityType = "contract"
	EntityAsset    EntityType = "asset"
	EntityFunction EntityType = "function"
	EntityTopic    EntityType = "topic"
	EntityAccount  EntityType = "account"
	EntityExplore  EntityType = "explore"
)

type Entity struct {
	Type      EntityType
	Name      string
	Subtitle  string
	Value     string
	Href      string
	Frequency int
	Recent    int
}

type Registry struct{ Entries []Entity }

func DefaultRegistry() *Registry {
	return &Registry{Entries: []Entity{
		{Type: EntityAsset, Name: "USDC", Subtitle: "Asset code", Value: "USDC", Href: "/v2/explore?asset=USDC", Frequency: 100},
		{Type: EntityAsset, Name: "XLM", Subtitle: "Native asset", Value: "XLM", Href: "/v2/explore?asset=XLM", Frequency: 100},
		{Type: EntityAsset, Name: "EURC", Subtitle: "Asset code", Value: "EURC", Href: "/v2/explore?asset=EURC", Frequency: 80},
		{Type: EntityFunction, Name: "swap", Subtitle: "Function call", Value: "swap", Href: "/v2/explore?fn=swap&topic=swap", Frequency: 95},
		{Type: EntityFunction, Name: "add_liquidity", Subtitle: "Function call", Value: "add_liquidity", Href: "/v2/explore?fn=add_liquidity", Frequency: 75},
		{Type: EntityFunction, Name: "approve", Subtitle: "Function call", Value: "approve", Href: "/v2/explore?fn=approve&topic=approve", Frequency: 70},
		{Type: EntityFunction, Name: "transfer", Subtitle: "Function call", Value: "transfer", Href: "/v2/explore?fn=transfer&topic=transfer", Frequency: 90},
		{Type: EntityTopic, Name: "transfer", Subtitle: "CAP/event topic", Value: "transfer", Href: "/v2/explore?topic=transfer", Frequency: 100},
		{Type: EntityTopic, Name: "swap", Subtitle: "Activity topic", Value: "swap", Href: "/v2/explore?topic=swap", Frequency: 90},
		{Type: EntityTopic, Name: "mint", Subtitle: "Activity topic", Value: "mint", Href: "/v2/explore?topic=mint", Frequency: 55},
		{Type: EntityTopic, Name: "burn", Subtitle: "Activity topic", Value: "burn", Href: "/v2/explore?topic=burn", Frequency: 45},
		{Type: EntityTopic, Name: "deposit", Subtitle: "Activity topic", Value: "deposit", Href: "/v2/explore?topic=deposit", Frequency: 55},
		{Type: EntityTopic, Name: "withdraw", Subtitle: "Activity topic", Value: "withdraw", Href: "/v2/explore?topic=withdraw", Frequency: 55},
		{Type: EntityContract, Name: "Soroswap router", Subtitle: "Known protocol contract", Value: "soroswap", Href: "/v2/explore?q=Soroswap", Frequency: 80},
		{Type: EntityContract, Name: "Phoenix AMM", Subtitle: "Known protocol", Value: "phoenix", Href: "/v2/explore?q=Phoenix", Frequency: 75},
		{Type: EntityContract, Name: "Blend pool", Subtitle: "Known lending protocol", Value: "blend", Href: "/v2/explore?q=Blend", Frequency: 70},
	}}
}

func (r *Registry) Search(input string, limit int) []Entity {
	if r == nil {
		r = DefaultRegistry()
	}
	q := strings.ToLower(strings.TrimSpace(input))
	if q == "" {
		return nil
	}
	type scored struct {
		e     Entity
		score int
	}
	var scoreds []scored
	for _, e := range r.Entries {
		name := strings.ToLower(e.Name)
		value := strings.ToLower(e.Value)
		score := 0
		switch {
		case name == q || value == q:
			score = 1000
		case strings.HasPrefix(name, q) || strings.HasPrefix(value, q):
			score = 700
		case strings.Contains(name, q) || strings.Contains(value, q):
			score = 400
		}
		if score == 0 {
			continue
		}
		score += e.Frequency + e.Recent
		scoreds = append(scoreds, scored{e, score})
	}
	sort.SliceStable(scoreds, func(i, j int) bool { return scoreds[i].score > scoreds[j].score })
	if limit <= 0 || limit > len(scoreds) {
		limit = len(scoreds)
	}
	out := make([]Entity, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, scoreds[i].e)
	}
	return out
}
