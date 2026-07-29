package handlers

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/intent"
	prismsearch "github.com/withObsrvr/prism/internal/search"
)

type entitySearchEvidence struct {
	Attempted             bool
	Status                string
	Entities              []prismsearch.Entity
	HasMore               bool
	Warnings              []string
	Source                string
	CompleteThroughLedger int64
}

func (h *Handlers) SearchSuggest(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	network := networkFromRequest(r)
	resolution, evidence := h.resolveSearchWithEntities(r.Context(), network, q)
	fmt.Fprint(w, `<div class="v2-suggest-panel">`)
	renderSearchResolutionSuggestion(w, resolution, true)

	seen := map[string]bool{resolution.Destination: true}
	remaining := 6
	for _, entity := range evidence.Entities {
		if remaining == 0 || seen[entity.Href] {
			continue
		}
		renderEntitySuggestion(w, entity)
		seen[entity.Href] = true
		remaining--
	}
	for _, entity := range prismsearch.DefaultRegistry().Search(q, remaining) {
		if remaining == 0 || seen[entity.Href] || suppressRegistryEntity(entity, evidence) {
			continue
		}
		renderEntitySuggestion(w, entity)
		seen[entity.Href] = true
		remaining--
	}
	renderEntitySearchEvidence(w, evidence, resolution)
	fmt.Fprintf(w, `<div class="v2-suggest-footer">Route <code>%s</code> · %.0f%% confidence</div>`, html.EscapeString(resolution.RuleID), resolution.Confidence*100)
	fmt.Fprint(w, `</div>`)
}

func (h *Handlers) SearchSubmit(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.FormValue("q"))
	if q == "" {
		q = strings.TrimSpace(r.URL.Query().Get("q"))
	}
	if q == "" {
		target := "/v2/explore"
		if qs := exploreFormQuery(r); qs != "" {
			target += "?" + qs
		}
		hxOrRedirect(w, r, target)
		return
	}
	resolution := h.resolveSearch(r.Context(), networkFromRequest(r), q)
	if exploreFormQuery(r) != "" {
		parsed, _ := prismsearch.Parse(q)
		mergeExploreForm(r, &parsed)
		if qs := parsed.QueryString(); qs != "" {
			hxOrRedirect(w, r, "/v2/explore?"+qs)
			return
		}
	}
	hxOrRedirect(w, r, resolution.Destination)
}

func (h *Handlers) resolveSearch(ctx context.Context, network, query string) prismsearch.SearchResolution {
	resolution, _ := h.resolveSearchWithEntities(ctx, network, query)
	return resolution
}

func (h *Handlers) resolveSearchWithEntities(ctx context.Context, network, query string) (prismsearch.SearchResolution, entitySearchEvidence) {
	ctx, cancel := context.WithTimeout(ctx, 1250*time.Millisecond)
	defer cancel()
	registry := intent.DefaultRegistry()
	resolver := prismsearch.NewResolver(prismsearch.DefaultRegistry())
	resolution := resolver.Resolve(query, prismsearch.ResolveOptions{MatchAnswer: func(input string) (prismsearch.AnswerCandidate, bool) {
		match, ok := registry.Match(input)
		if !ok {
			return prismsearch.AnswerCandidate{}, false
		}
		return prismsearch.AnswerCandidate{
			RuleID:     string(match.ID),
			Label:      searchAnswerLabel(match),
			Summary:    questionIntentSummary(match),
			Confidence: match.Confidence,
			Slots:      match.Slots,
		}, true
	}})
	evidence := entitySearchEvidence{}
	if shouldLookupDynamicEntities(resolution) {
		evidence = h.dynamicSearchEvidenceFiltered(ctx, network, query, 7, entitySearchTypeFilters(resolution))
		resolution = reconcileGatewayResolution(query, resolution, evidence)
	}
	if resolution.Kind == prismsearch.SearchOpen && strings.HasPrefix(resolution.Destination, "/v2/contract/") {
		entityValue := resolution.Slots["entity_value"]
		if entityValue != "" {
			if destination := h.detectSmartRedirect(ctx, network, entityValue); destination != "" {
				resolution.Destination = destination
				resolution.ActionLabel = gatewayEntityActionLabel(prismsearch.Entity{Type: prismsearch.EntityContract, EntityKind: resolution.Slots["entity_type"]}, destination)
			}
		}
	}
	return resolution, evidence
}

func renderSearchResolutionSuggestion(w http.ResponseWriter, resolution prismsearch.SearchResolution, primary bool) {
	className := "v2-suggest-row"
	if primary {
		className += " primary"
	}
	if resolution.Kind == prismsearch.SearchUnsupported {
		className += " unsupported"
	}
	fmt.Fprintf(w, `<a class="%s" href="%s"><span><b>%s</b><small>%s</small></span><strong>%s</strong></a>`,
		html.EscapeString(className), html.EscapeString(resolution.Destination), html.EscapeString(resolution.ActionLabel), html.EscapeString(resolution.Summary), html.EscapeString(searchKindLabel(resolution.Kind)))
}

func renderEntitySuggestion(w http.ResponseWriter, entity prismsearch.Entity) {
	verb := entityActionVerb(entity)
	fmt.Fprintf(w, `<a class="v2-suggest-row" href="%s"><span><b>%s %s</b><small>%s</small>`,
		html.EscapeString(entity.Href), verb, html.EscapeString(entity.Name), html.EscapeString(entity.Subtitle))
	if entity.MatchType != "" || entity.VerificationStatus != "" {
		fmt.Fprint(w, `<small class="v2-suggest-evidence-line">`)
		if match := entityMatchLabel(entity); match != "" {
			fmt.Fprintf(w, `<span>%s</span>`, html.EscapeString(match))
		}
		if identity := entityIdentityLabel(entity); identity != "" {
			fmt.Fprintf(w, `<span>%s</span>`, html.EscapeString(identity))
		}
		fmt.Fprint(w, `</small>`)
	}
	fmt.Fprintf(w, `</span><em>%s</em></a>`, verb)
}

func (h *Handlers) dynamicSearchEntities(ctx context.Context, network, query string, limit int) []prismsearch.Entity {
	return h.dynamicSearchEvidence(ctx, network, query, limit).Entities
}

func (h *Handlers) dynamicSearchEvidence(ctx context.Context, network, query string, limit int) entitySearchEvidence {
	return h.dynamicSearchEvidenceFiltered(ctx, network, query, limit, nil)
}

func (h *Handlers) dynamicSearchEvidenceFiltered(ctx context.Context, network, query string, limit int, typeFilters []string) entitySearchEvidence {
	query = strings.TrimSpace(query)
	if h.Gateway == nil || len([]rune(query)) < 2 || limit <= 0 || h.DataSource == "mock" {
		return entitySearchEvidence{}
	}
	evidence := entitySearchEvidence{Attempted: true, Status: "unavailable"}
	ctx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	results, err := h.Gateway.SearchEntities(ctx, network, query, gateway.SearchOptions{Limit: limit, Types: typeFilters})
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn("dynamic search suggestions unavailable", "network", network, "error", err)
		}
		evidence.Warnings = []string{"Prism could not verify the live entity-search response."}
		return evidence
	}
	evidence.Status = results.Status
	evidence.HasMore = results.HasMore
	evidence.Warnings = append(evidence.Warnings, results.Warnings...)
	evidence.Source = results.Provenance.Source
	evidence.CompleteThroughLedger = results.Provenance.CompleteThroughLedger
	if results.Status == "unavailable" {
		return evidence
	}
	entities := make([]prismsearch.Entity, 0, min(limit, len(results.Results)))
	for _, result := range results.Results {
		entity, ok := searchEntityFromGateway(result)
		if !ok {
			continue
		}
		entities = append(entities, entity)
		if len(entities) == limit {
			break
		}
	}
	evidence.Entities = entities
	return evidence
}

func entitySearchTypeFilters(resolution prismsearch.SearchResolution) []string {
	if isEntityLikeResolution(resolution) {
		return []string{"asset", "sac"}
	}
	if resolution.Entity.Known() {
		switch resolution.Entity.Type {
		case prismsearch.ClassAccount:
			return []string{"account"}
		case prismsearch.ClassContract:
			return []string{"contract", "sac", "protocol_contract"}
		case prismsearch.ClassAsset:
			return []string{"asset", "sac"}
		case prismsearch.ClassTxHash:
			return []string{"transaction"}
		case prismsearch.ClassLedger:
			return []string{"ledger"}
		}
	}
	if resolution.RuleID == "registry.contract" {
		return []string{"protocol", "protocol_contract", "contract"}
	}
	return nil
}

func searchEntityFromGateway(result gateway.SearchResult) (prismsearch.Entity, bool) {
	id := strings.TrimSpace(result.ID)
	if id == "" {
		return prismsearch.Entity{}, false
	}
	label := firstNonEmpty(strings.TrimSpace(result.DisplayName), strings.TrimSpace(result.Label), strings.TrimSpace(result.Symbol))
	if label == "" {
		label = id
	}
	kind := strings.ToLower(strings.TrimSpace(result.EntityKind))
	if kind == "" {
		kind = strings.ToLower(strings.TrimSpace(result.Type))
	}
	if (kind == "classic_asset" || kind == "sac") && result.Symbol != "" &&
		(strings.EqualFold(label, id) || strings.EqualFold(label, result.CanonicalSlug) || strings.Contains(label, ":")) {
		label = strings.TrimSpace(result.Symbol)
	}
	entity := prismsearch.Entity{
		Name:               label,
		Value:              id,
		Subtitle:           gatewayEntitySubtitle(result, kind),
		EntityKind:         kind,
		CanonicalSlug:      strings.TrimSpace(result.CanonicalSlug),
		Symbol:             strings.TrimSpace(result.Symbol),
		MatchedField:       strings.TrimSpace(result.MatchedField),
		MatchType:          strings.ToLower(strings.TrimSpace(result.MatchType)),
		IdentitySource:     strings.TrimSpace(result.IdentitySource),
		VerificationStatus: strings.ToLower(strings.TrimSpace(result.VerificationStatus)),
	}
	entity.SACContractID, _ = searchResultStringDetail(result, "sac_contract_id")
	switch kind {
	case "account":
		entity.Type, entity.Href = prismsearch.EntityAccount, "/v2/account/"+url.PathEscape(id)
	case "smart_wallet":
		entity.Type, entity.Href = prismsearch.EntityAccount, "/v2/account/"+url.PathEscape(id)+"/smart"
	case "contract":
		entity.Type, entity.Href = prismsearch.EntityContract, "/v2/contract/"+url.PathEscape(id)
	case "protocol_contract":
		if strings.HasPrefix(strings.ToUpper(id), "C") {
			entity.Type, entity.Href = prismsearch.EntityContract, "/v2/contract/"+url.PathEscape(id)
		} else {
			entity.Type, entity.Href = prismsearch.EntityExplore, "/v2/explore?q="+url.QueryEscape(id)
		}
	case "protocol":
		entity.Type, entity.Href = prismsearch.EntityExplore, "/v2/explore?q="+url.QueryEscape(firstNonEmpty(result.DisplayName, result.Label, result.CanonicalSlug, id))
	case "liquidity_pool":
		entity.Type, entity.Href = prismsearch.EntityExplore, "/v2/explore?q="+url.QueryEscape(id)
	case "transaction", "tx":
		entity.Type, entity.Href = prismsearch.EntityExplore, "/v2/tx/"+url.PathEscape(id)
	case "ledger":
		entity.Type, entity.Href = prismsearch.EntityExplore, "/v2/ledger/"+url.PathEscape(id)
	case "classic_asset", "asset":
		slug := assetSearchSlug(result)
		if slug == "" {
			slug = id
		}
		entity.Type, entity.Href = prismsearch.EntityAsset, "/v2/assets/"+url.PathEscape(slug)
	case "sac", "token":
		slug := assetSearchSlug(result)
		if slug == "" {
			slug = id
		}
		entity.Type, entity.Href = prismsearch.EntityAsset, "/v2/assets/"+url.PathEscape(slug)
	default:
		return prismsearch.Entity{}, false
	}
	return entity, true
}

func exactGatewayEntity(query string, entities []prismsearch.Entity) (prismsearch.Entity, bool) {
	query = strings.TrimSpace(query)
	exact := make([]prismsearch.Entity, 0, len(entities))
	for _, entity := range entities {
		if entity.MatchType == "exact" || (entity.MatchType == "" && (strings.EqualFold(query, entity.Name) || strings.EqualFold(query, entity.Value))) {
			exact = append(exact, entity)
		}
	}
	if len(exact) == 1 {
		return exact[0], true
	}
	if len(exact) == 0 && len(entities) == 1 && entities[0].MatchType == "prefix" {
		return entities[0], true
	}
	return prismsearch.Entity{}, false
}

func shouldLookupDynamicEntities(resolution prismsearch.SearchResolution) bool {
	switch resolution.Kind {
	case prismsearch.SearchOpen, prismsearch.SearchUnsupported:
		return resolution.RuleID != "unsupported.no_answer_rule"
	case prismsearch.SearchExplore:
		return strings.HasPrefix(resolution.RuleID, "registry.") || isSingleAssetActivityResolution(resolution)
	default:
		return false
	}
}

func reconcileGatewayResolution(query string, resolution prismsearch.SearchResolution, evidence entitySearchEvidence) prismsearch.SearchResolution {
	if !evidence.Attempted {
		return resolution
	}
	if evidence.Status == "unavailable" {
		if isEntityLikeResolution(resolution) {
			return unavailableEntityResolution(query)
		}
		return resolution
	}
	if evidence.Status == "partial" && len(evidence.Entities) == 0 && isEntityLikeResolution(resolution) {
		return partialEntityResolution(query)
	}

	exact := make([]prismsearch.Entity, 0, len(evidence.Entities))
	for _, entity := range evidence.Entities {
		if entity.MatchType == "exact" {
			exact = append(exact, entity)
		}
	}
	if len(exact) == 1 {
		return gatewayEntityResolution(query, exact[0])
	}
	if len(exact) > 1 {
		return ambiguousGatewayResolution(query, exact, evidence.HasMore)
	}
	if entity, ok := exactGatewayEntity(query, evidence.Entities); ok {
		return gatewayEntityResolution(query, entity)
	}

	if len(evidence.Entities) > 0 && (resolution.Kind == prismsearch.SearchUnsupported || isEntityLikeResolution(resolution)) {
		resolution.Kind = prismsearch.SearchUnsupported
		resolution.ActionLabel = "No exact match"
		resolution.Summary = "Prism found close live entity matches. Choose one below; it will not open a fuzzy match automatically."
		resolution.RuleID = "gateway.entity.close_matches"
		resolution.Confidence = 0
		resolution.Destination = "/v2/search/unsupported?q=" + url.QueryEscape(strings.TrimSpace(query))
		return resolution
	}
	if len(evidence.Entities) == 0 && isEntityLikeResolution(resolution) {
		resolution.Kind = prismsearch.SearchUnsupported
		resolution.ActionLabel = "No live entity match"
		resolution.Summary = "The current live identity index returned no match, so Prism did not guess an entity."
		resolution.RuleID = "gateway.entity.authoritative_empty"
		resolution.Confidence = 0
		resolution.Destination = "/v2/search/unsupported?q=" + url.QueryEscape(strings.TrimSpace(query))
	}
	return resolution
}

func gatewayEntityResolution(query string, entity prismsearch.Entity) prismsearch.SearchResolution {
	confidence := 1.0
	matchSummary := entityMatchLabel(entity)
	if entity.MatchType == "prefix" {
		confidence = 0.85
		matchSummary += " and the only returned candidate"
	}
	identitySummary := entityIdentityLabel(entity)
	summary := strings.TrimSuffix(entity.Subtitle, ".") + "."
	if matchSummary != "" {
		summary += " " + matchSummary + "."
	}
	if identitySummary != "" {
		summary += " " + identitySummary + "."
	}
	kind := prismsearch.SearchOpen
	if entityActionVerb(entity) == "Explore" {
		kind = prismsearch.SearchExplore
	}
	return prismsearch.SearchResolution{
		Kind:        kind,
		Query:       strings.TrimSpace(query),
		ActionLabel: gatewayEntityActionLabel(entity, entity.Href),
		Summary:     summary,
		RuleID:      "gateway.entity." + firstNonEmpty(entity.EntityKind, string(entity.Type)),
		Confidence:  confidence,
		Destination: entity.Href,
		Slots: map[string]string{
			"entity_type":  firstNonEmpty(entity.EntityKind, string(entity.Type)),
			"entity_value": entity.Value,
		},
	}
}

func ambiguousGatewayResolution(query string, entities []prismsearch.Entity, hasMore bool) prismsearch.SearchResolution {
	countLabel := fmt.Sprintf("%d exact live matches", len(entities))
	if hasMore {
		countLabel = fmt.Sprintf("at least %d exact live matches", len(entities))
	}
	trimmed := strings.TrimSpace(query)
	destination := "/v2/explore?q=" + url.QueryEscape(trimmed)
	action := "Explore matching activity"
	if allGatewayEntitiesAreAssets(entities) {
		destination = "/v2/explore?asset=" + url.QueryEscape(strings.ToUpper(trimmed))
		action = "Explore all " + strings.ToUpper(trimmed) + " activity"
	}
	return prismsearch.SearchResolution{
		Kind:        prismsearch.SearchExplore,
		Query:       trimmed,
		ActionLabel: action,
		Summary:     "Live search found " + countLabel + ". Choose a specific identity below; Prism will not select one implicitly.",
		RuleID:      "gateway.entity.ambiguous",
		Confidence:  1,
		Destination: destination,
		Slots:       map[string]string{"entity_type": "ambiguous", "entity_value": trimmed},
	}
}

func unavailableEntityResolution(query string) prismsearch.SearchResolution {
	trimmed := strings.TrimSpace(query)
	return prismsearch.SearchResolution{
		Kind:        prismsearch.SearchUnsupported,
		Query:       trimmed,
		ActionLabel: "Entity lookup unavailable",
		Summary:     "Prism cannot verify this entity while the live identity index is unavailable, so it did not guess.",
		RuleID:      "gateway.entity.unavailable",
		Confidence:  0,
		Destination: "/v2/search/unsupported?q=" + url.QueryEscape(trimmed),
		Slots:       map[string]string{},
	}
}

func partialEntityResolution(query string) prismsearch.SearchResolution {
	trimmed := strings.TrimSpace(query)
	return prismsearch.SearchResolution{
		Kind:        prismsearch.SearchUnsupported,
		Query:       trimmed,
		ActionLabel: "Entity lookup incomplete",
		Summary:     "The live identity index was readable, but optional lookups did not complete. Prism did not treat that as no match.",
		RuleID:      "gateway.entity.partial",
		Confidence:  0,
		Destination: "/v2/search/unsupported?q=" + url.QueryEscape(trimmed),
		Slots:       map[string]string{},
	}
}

func isRegistryIdentityResolution(resolution prismsearch.SearchResolution) bool {
	return resolution.RuleID == "registry.asset"
}

func isSingleAssetActivityResolution(resolution prismsearch.SearchResolution) bool {
	return resolution.RuleID == "activity.structured" &&
		resolution.Activity.Asset != "" &&
		resolution.Activity.Topic == "" &&
		resolution.Activity.Fn == "" &&
		resolution.Activity.Status == "" &&
		strings.EqualFold(strings.TrimSpace(resolution.Query), resolution.Activity.Asset)
}

func isEntityLikeResolution(resolution prismsearch.SearchResolution) bool {
	return isRegistryIdentityResolution(resolution) || isSingleAssetActivityResolution(resolution)
}

func allGatewayEntitiesAreAssets(entities []prismsearch.Entity) bool {
	if len(entities) == 0 {
		return false
	}
	for _, entity := range entities {
		if entity.Type != prismsearch.EntityAsset {
			return false
		}
	}
	return true
}

func suppressRegistryEntity(entity prismsearch.Entity, evidence entitySearchEvidence) bool {
	return evidence.Attempted && entity.Type == prismsearch.EntityAsset
}

func renderEntitySearchEvidence(w http.ResponseWriter, evidence entitySearchEvidence, resolution prismsearch.SearchResolution) {
	if !evidence.Attempted {
		return
	}
	switch evidence.Status {
	case "unavailable":
		fmt.Fprint(w, `<div class="v2-suggest-state unavailable" role="status"><b>Live entity lookup unavailable</b><small>Prism did not guess at entity identities. Try again shortly.</small></div>`)
	case "partial":
		fmt.Fprint(w, `<div class="v2-suggest-state partial" role="status"><b>Some live matches may be missing</b><small>The identity index was readable, but one or more optional lookups did not complete.</small></div>`)
	case "ready":
		if len(evidence.Entities) == 0 && resolution.Kind == prismsearch.SearchUnsupported {
			fmt.Fprint(w, `<div class="v2-suggest-state empty" role="status"><b>No live entity matches</b><small>This is an authoritative empty result for the current identity snapshot.</small></div>`)
		}
	}
	if len(evidence.Warnings) > 0 {
		fmt.Fprintf(w, `<div class="v2-suggest-warning">Evidence note: %s</div>`, html.EscapeString(friendlySearchWarning(evidence.Warnings[0])))
	}
	if evidence.HasMore {
		fmt.Fprint(w, `<div class="v2-suggest-state more" role="status"><b>More matches exist</b><small>Results are capped. Add an issuer, entity type, or a longer ID prefix to narrow the list.</small></div>`)
	}
	if evidence.Status == "ready" || evidence.Status == "partial" {
		ledger := "current serving snapshot"
		if evidence.CompleteThroughLedger > 0 {
			ledger = "ledger " + gateway.FormatNumber(evidence.CompleteThroughLedger)
		}
		fmt.Fprintf(w, `<div class="v2-suggest-provenance">Live identity index through %s`, html.EscapeString(ledger))
		if source := friendlyIdentitySource(evidence.Source); source != "" {
			fmt.Fprintf(w, ` · %s`, html.EscapeString(source))
		}
		fmt.Fprint(w, `</div>`)
	}
}

func friendlySearchWarning(warning string) string {
	normalized := strings.ToLower(strings.TrimSpace(warning))
	switch {
	case strings.Contains(normalized, "no complete source watermark"):
		return "The identity index has not published a complete network snapshot yet."
	case strings.Contains(normalized, "could not verify"):
		return "Prism could not verify the live entity-search response."
	default:
		return strings.TrimSpace(warning)
	}
}

func entityActionVerb(entity prismsearch.Entity) string {
	if strings.Contains(entity.Href, "/explore") {
		return "Explore"
	}
	return "Open"
}

func entityMatchLabel(entity prismsearch.Entity) string {
	matchType := strings.ToLower(strings.TrimSpace(entity.MatchType))
	if matchType == "" {
		return ""
	}
	field := friendlyMatchedField(entity.MatchedField)
	if field == "" {
		return strings.ToUpper(matchType[:1]) + matchType[1:] + " match"
	}
	return strings.ToUpper(matchType[:1]) + matchType[1:] + " " + field + " match"
}

func entityIdentityLabel(entity prismsearch.Entity) string {
	status := strings.ToLower(strings.TrimSpace(entity.VerificationStatus))
	label := ""
	switch status {
	case "verified":
		label = "Verified identity"
	case "unverified":
		label = "Unverified identity"
	case "inferred":
		label = "Inferred identity"
	}
	if source := friendlyIdentitySource(entity.IdentitySource); source != "" {
		if label == "" {
			return source
		}
		return label + " · " + source
	}
	return label
}

func friendlyMatchedField(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "entity_id":
		return "ID"
	case "canonical_slug":
		return "canonical name"
	case "symbol":
		return "symbol"
	case "display_name":
		return "name"
	case "search_terms":
		return "name or symbol"
	default:
		return ""
	}
}

func friendlyIdentitySource(source string) string {
	normalized := strings.ToLower(strings.TrimSpace(source))
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "entity_search"):
		return "serving-only index"
	case strings.Contains(normalized, "asset"):
		return "asset registry"
	case strings.Contains(normalized, "token"):
		return "token registry"
	case strings.Contains(normalized, "liquidity_pool"):
		return "liquidity pool registry"
	case strings.Contains(normalized, "account"):
		return "account registry"
	case strings.Contains(normalized, "transaction"):
		return "recent transaction index"
	case strings.Contains(normalized, "ledger"):
		return "recent ledger index"
	case strings.Contains(normalized, "contract"):
		return "contract registry"
	default:
		return "identity registry"
	}
}

func gatewayEntitySubtitle(result gateway.SearchResult, kind string) string {
	switch kind {
	case "classic_asset", "asset":
		sac := ""
		if contractID, ok := searchResultStringDetail(result, "sac_contract_id"); ok {
			sac = " · SAC " + gateway.ShortAddress(contractID)
		}
		if issuer, ok := searchResultStringDetail(result, "asset_issuer", "issuer"); ok {
			return "Classic asset · issuer " + gateway.ShortAddress(issuer) + sac
		}
		return "Classic asset" + sac
	case "sac", "token":
		if slug := strings.TrimSpace(result.CanonicalSlug); slug != "" {
			return "Stellar Asset Contract · " + slug
		}
		return "Stellar Asset Contract"
	case "liquidity_pool":
		if pair, ok := searchResultStringDetail(result, "asset_pair"); ok {
			return "Liquidity pool · " + pair
		}
		return "Liquidity pool"
	case "protocol":
		return "Protocol"
	case "protocol_contract":
		return "Protocol contract · " + gateway.ShortAddress(result.ID)
	case "contract":
		return "Contract · " + gateway.ShortAddress(result.ID)
	case "smart_wallet":
		return "Smart account · " + gateway.ShortAddress(result.ID)
	case "account":
		if domain, ok := searchResultStringDetail(result, "home_domain"); ok {
			return "Account · " + domain
		}
		return "Account · " + gateway.ShortAddress(result.ID)
	case "transaction", "tx":
		return "Transaction · " + gateway.ShortHash(result.ID)
	case "ledger":
		return "Ledger #" + result.ID
	default:
		return "Live entity"
	}
}

func gatewayEntityActionLabel(entity prismsearch.Entity, destination string) string {
	switch {
	case strings.Contains(destination, "/smart"):
		return "Open smart account"
	case strings.Contains(destination, "/tx/"):
		return "Open transaction"
	case strings.Contains(destination, "/ledger/"):
		return "Open ledger"
	case entity.EntityKind == "liquidity_pool":
		return "Explore liquidity pool"
	case entity.EntityKind == "protocol":
		return "Explore protocol"
	case entity.EntityKind == "sac":
		return "Open linked asset"
	case entity.Type == prismsearch.EntityContract:
		return "Open contract"
	case entity.Type == prismsearch.EntityAccount:
		return "Open account"
	case entity.Type == prismsearch.EntityAsset:
		return "Open asset"
	default:
		return "Open entity"
	}
}

func exploreFormQuery(r *http.Request) string {
	v := url.Values{}
	for _, key := range []string{"scope", "topic", "fn", "asset", "time", "status"} {
		value := strings.TrimSpace(r.FormValue(key))
		if value == "" {
			continue
		}
		if key == "scope" && value == "all" {
			continue
		}
		if key == "time" && value == "1h" {
			continue
		}
		v.Set(key, value)
	}
	return v.Encode()
}

func mergeExploreForm(r *http.Request, q *prismsearch.Query) {
	if v := strings.TrimSpace(r.FormValue("scope")); v != "" {
		q.Scope = v
	}
	if v := strings.TrimSpace(r.FormValue("topic")); v != "" {
		q.Topic = v
	}
	if v := strings.TrimSpace(r.FormValue("fn")); v != "" {
		q.Fn = v
	}
	if v := strings.TrimSpace(r.FormValue("asset")); v != "" {
		q.Asset = strings.ToUpper(v)
	}
	if v := strings.TrimSpace(r.FormValue("time")); v != "" {
		q.Time = v
	}
	if v := strings.TrimSpace(r.FormValue("status")); v != "" {
		q.Status = v
	}
}

func hxOrRedirect(w http.ResponseWriter, r *http.Request, target string) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func questionIntentSummary(m intent.Match) string {
	switch m.ID {
	case intent.ExpiringContracts:
		return "Current contract TTL attention snapshot with evidence."
	case intent.ProtocolBusy:
		return "Known protocol activity using an explicit evidence window."
	case intent.TransactionFailure:
		return "Decoded receipt and semantic failure evidence for the supplied transaction."
	case intent.ContractActivity:
		return "Contract analytics with the returned evidence window disclosed."
	case intent.AssetActivity:
		return "Asset transfers, accounts, holders, and volume from the 24-hour record."
	case intent.RecentFailures:
		return "Recent decoded failures with collection coverage disclosed."
	default:
		return "Deterministic answer with linked evidence."
	}
}

func searchAnswerLabel(match intent.Match) string {
	switch match.ID {
	case intent.TransactionFailure:
		return "Answer transaction failure"
	case intent.ContractActivity:
		return "Answer contract activity"
	case intent.AssetActivity:
		return "Answer asset activity"
	case intent.RecentFailures:
		return "Answer recent failures"
	case intent.ExpiringContracts:
		return "Answer contract TTL question"
	case intent.ProtocolBusy:
		return "Answer protocol activity"
	default:
		return "Answer supported question"
	}
}

func searchOpenLabel(entity prismsearch.Classification, destination string) string {
	switch {
	case strings.Contains(destination, "/smart"):
		return "Open smart account"
	case strings.Contains(destination, "/assets/"):
		return "Open asset"
	case entity.Type == prismsearch.ClassTxHash:
		return "Open transaction"
	case entity.Type == prismsearch.ClassAccount:
		return "Open account"
	case entity.Type == prismsearch.ClassContract:
		return "Open contract"
	case entity.Type == prismsearch.ClassLedger:
		return "Open ledger"
	default:
		return "Open entity"
	}
}

func searchKindLabel(kind prismsearch.SearchResolutionKind) string {
	switch kind {
	case prismsearch.SearchOpen:
		return "Open"
	case prismsearch.SearchExplore:
		return "Explore"
	case prismsearch.SearchAnswer:
		return "Answer"
	default:
		return "Unsupported"
	}
}
