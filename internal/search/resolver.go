package search

import (
	"net/url"
	"strings"
)

type SearchResolutionKind string

const (
	SearchOpen        SearchResolutionKind = "open"
	SearchExplore     SearchResolutionKind = "explore"
	SearchAnswer      SearchResolutionKind = "answer"
	SearchUnsupported SearchResolutionKind = "unsupported"
)

type AnswerCandidate struct {
	RuleID     string
	Label      string
	Summary    string
	Confidence float64
	Slots      map[string]string
}

type AnswerMatcher func(input string) (AnswerCandidate, bool)

type ResolveOptions struct {
	MatchAnswer AnswerMatcher
}

type SearchResolution struct {
	Kind           SearchResolutionKind
	Query          string
	ActionLabel    string
	Summary        string
	RuleID         string
	Confidence     float64
	Destination    string
	Slots          map[string]string
	Entity         Classification
	Activity       Query
	EmbeddedEntity bool
}

type Resolver struct {
	Registry *Registry
}

func NewResolver(registry *Registry) *Resolver {
	if registry == nil {
		registry = DefaultRegistry()
	}
	return &Resolver{Registry: registry}
}

func (r *Resolver) Resolve(input string, options ResolveOptions) SearchResolution {
	query := strings.TrimSpace(input)
	if query == "" {
		return unsupportedResolution(query, "Prism needs a transaction, account, contract, asset, ledger, or supported activity phrase.")
	}

	exact := Classify(query)
	if exact.Known() {
		return openEntityResolution(query, exact, false)
	}

	embedded := ExtractIdentifier(query)
	answer, hasAnswer := matchAnswer(query, options.MatchAnswer)
	if embedded.Known() {
		if hasAnswer {
			return answerResolution(query, answer)
		}
		return openEntityResolution(query, embedded, true)
	}
	if hasAnswer {
		return answerResolution(query, answer)
	}

	registry := r.Registry
	if registry == nil {
		registry = DefaultRegistry()
	}
	if matches := registry.Search(query, 1); len(matches) == 1 && entityExactlyMatches(matches[0], query) {
		return registryResolution(query, matches[0])
	}
	if openQuestion(query) {
		resolution := unsupportedResolution(query, "Prism recognized question wording, but no deterministic answer rule matched an unambiguous evidence subject.")
		resolution.RuleID = "unsupported.no_answer_rule"
		return resolution
	}

	activity, confidence := Parse(query)
	if confidence >= 0.45 && hasStructuredActivity(activity) {
		return SearchResolution{
			Kind:        SearchExplore,
			Query:       query,
			ActionLabel: "Explore " + activityLabel(activity),
			Summary:     activitySummary(activity),
			RuleID:      "activity.structured",
			Confidence:  confidence,
			Destination: "/v2/explore?" + activity.QueryString(),
			Slots:       activitySlots(activity),
			Activity:    activity,
		}
	}

	return unsupportedResolution(query, "No supported identifier, activity filter, or deterministic answer rule matched this query.")
}

func openQuestion(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, prefix := range []string{"is ", "are ", "was ", "were ", "how ", "what ", "which ", "why ", "did ", "do ", "does ", "can ", "could ", "should "} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func registryResolution(query string, entity Entity) SearchResolution {
	kind := SearchOpen
	verb := "Open"
	if strings.Contains(entity.Href, "/explore") {
		kind = SearchExplore
		verb = "Explore"
	}
	return SearchResolution{
		Kind:        kind,
		Query:       query,
		ActionLabel: verb + " " + entity.Name,
		Summary:     entity.Subtitle + ", matched by Prism's known-entity registry.",
		RuleID:      "registry." + string(entity.Type),
		Confidence:  0.8,
		Destination: entity.Href,
		Slots:       map[string]string{"entity_type": string(entity.Type), "entity_value": entity.Value},
	}
}

func matchAnswer(query string, matcher AnswerMatcher) (AnswerCandidate, bool) {
	if matcher == nil {
		return AnswerCandidate{}, false
	}
	answer, ok := matcher(query)
	return answer, ok && answer.RuleID != "" && answer.Confidence >= 0.65
}

func openEntityResolution(query string, entity Classification, embedded bool) SearchResolution {
	label := entityLabel(entity.Type)
	summary := "Recognized an exact " + label + " identifier."
	if embedded {
		summary = "Found a " + label + " identifier inside the query."
	}
	return SearchResolution{
		Kind:           SearchOpen,
		Query:          query,
		ActionLabel:    "Open " + label,
		Summary:        summary,
		RuleID:         "entity." + string(entity.Type),
		Confidence:     1,
		Destination:    entity.URL(),
		Slots:          map[string]string{"entity_type": string(entity.Type), "entity_value": entity.Value},
		Entity:         entity,
		EmbeddedEntity: embedded,
	}
}

func answerResolution(query string, answer AnswerCandidate) SearchResolution {
	label := strings.TrimSpace(answer.Label)
	if label == "" {
		label = "Answer supported question"
	}
	return SearchResolution{
		Kind:        SearchAnswer,
		Query:       query,
		ActionLabel: label,
		Summary:     answer.Summary,
		RuleID:      "intent." + answer.RuleID,
		Confidence:  answer.Confidence,
		Destination: "/v2/ask?q=" + url.QueryEscape(query),
		Slots:       cloneSlots(answer.Slots),
	}
}

func unsupportedResolution(query, summary string) SearchResolution {
	return SearchResolution{
		Kind:        SearchUnsupported,
		Query:       query,
		ActionLabel: "Unsupported question",
		Summary:     summary,
		RuleID:      "unsupported.no_supported_rule",
		Destination: "/v2/search/unsupported?q=" + url.QueryEscape(query),
		Slots:       map[string]string{},
	}
}

func hasStructuredActivity(query Query) bool {
	return query.Topic != "" || query.Fn != "" || query.Asset != "" || query.Status != "" || (query.Scope != "" && query.Scope != "all") || (query.Time != "" && query.Time != "1h")
}

func activityLabel(query Query) string {
	parts := make([]string, 0, 4)
	if query.Asset != "" {
		parts = append(parts, query.Asset)
	}
	if query.Topic != "" {
		parts = append(parts, pluralActivity(query.Topic))
	} else if query.Fn != "" {
		parts = append(parts, query.Fn+" calls")
	} else if query.Status == "failed" {
		parts = append(parts, "failed activity")
	} else {
		parts = append(parts, "activity")
	}
	if query.Time != "" {
		parts = append(parts, activityTimeLabel(query.Time))
	}
	return strings.Join(parts, " ")
}

func activitySummary(query Query) string {
	parts := []string{"Structured activity query"}
	if query.Scope != "" && query.Scope != "all" {
		parts = append(parts, query.Scope)
	}
	if query.Topic != "" {
		parts = append(parts, "topic "+query.Topic)
	}
	if query.Fn != "" {
		parts = append(parts, "function "+query.Fn)
	}
	if query.Asset != "" {
		parts = append(parts, "asset "+query.Asset)
	}
	if query.Status != "" {
		parts = append(parts, "status "+query.Status)
	}
	parts = append(parts, activityTimeLabel(query.Time))
	return strings.Join(parts, ", ") + "."
}

func activitySlots(query Query) map[string]string {
	return map[string]string{
		"scope": query.Scope, "topic": query.Topic, "function": query.Fn,
		"asset": query.Asset, "time": query.Time, "status": query.Status,
	}
}

func activityTimeLabel(value string) string {
	switch value {
	case "24h":
		return "last 24 hours"
	case "7d":
		return "last 7 days"
	case "30d":
		return "last 30 days"
	default:
		return "last hour"
	}
}

func pluralActivity(value string) string {
	switch value {
	case "swap":
		return "swaps"
	case "transfer":
		return "transfers"
	default:
		return value
	}
}

func entityLabel(value ClassType) string {
	switch value {
	case ClassTxHash:
		return "transaction"
	case ClassAccount:
		return "account"
	case ClassContract:
		return "contract"
	case ClassAsset:
		return "asset"
	case ClassMuxed:
		return "muxed account"
	case ClassLedger:
		return "ledger"
	case ClassFederation:
		return "federation address"
	default:
		return "entity"
	}
}

func entityExactlyMatches(entity Entity, query string) bool {
	query = strings.TrimSpace(query)
	return strings.EqualFold(entity.Name, query) || strings.EqualFold(entity.Value, query)
}

func cloneSlots(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
