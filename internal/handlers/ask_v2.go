package handlers

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/withObsrvr/prism/internal/intent"
)

func (h *Handlers) AskV2(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		hxOrRedirect(w, r, "/v2/home")
		return
	}
	if redirect := h.detectSmartRedirect(r.Context(), networkFromRequest(r), q); redirect != "" {
		hxOrRedirect(w, r, redirect)
		return
	}
	reg := intent.DefaultRegistry()
	match, ok := reg.Match(q)
	if !ok || match.Confidence < 0.65 {
		hxOrRedirect(w, r, "/v2/explore?q="+url.QueryEscape(q))
		return
	}
	result, err := reg.Execute(r.Context(), intent.Env{Gateway: h.Gateway, Network: networkFromRequest(r)}, match)
	if err != nil {
		h.Logger.Warn("ask intent execution failed", "query", q, "intent", match.ID, "error", err)
		result = intent.Result{
			Title:      "I could not answer that yet",
			Answer:     "The question matched a deterministic intent, but the required data was not available. Try exploring the raw activity instead.",
			Confidence: match.Confidence,
			Actions:    []intent.ActionLink{{Label: "Explore activity", Href: "/v2/explore?q=" + url.QueryEscape(q)}},
			Warnings:   []string{err.Error()},
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderAskPage(w, q, match, result, networkFromRequest(r))
}

func renderAskPage(w http.ResponseWriter, q string, m intent.Match, res intent.Result, network string) {
	fmt.Fprint(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Ask Prism</title><link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin><link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500;600&family=Source+Serif+4:opsz,wght@8..60,400;8..60,500;8..60,600&display=swap" rel="stylesheet"><link rel="stylesheet" href="/static/css/v2-unified.css"></head><body><div class="px-ask-app">`)
	renderAskTop(w, network)
	fmt.Fprint(w, `<nav class="px-tx-crumbs"><div class="px-tx-crumbs-inner"><a href="/v2/home">Home</a><span class="sep">/</span><span class="cur">Ask</span></div></nav>`)
	fmt.Fprint(w, `<div class="px-ask-page"><div class="px-ask-page-inner"><main class="px-ask-main">`)
	renderAskBar(w, q)
	renderAskInterpret(w, q, m)
	renderAskAnswer(w, m, res)
	renderAskNext(w, res)
	fmt.Fprint(w, `</main>`)
	renderAskSide(w, m, res)
	fmt.Fprint(w, `</div></div></div></body></html>`)
}

func renderAskTop(w http.ResponseWriter, network string) {
	fmt.Fprint(w, `<header class="ph-top"><div class="ph-top-inner"><a class="ph-brand" href="/v2/home"><span class="ph-mark"><svg width="22" height="22" viewBox="0 0 22 22" fill="none"><path d="M11 2 L19 7 L19 15 L11 20 L3 15 L3 7 Z" stroke="currentColor" stroke-width="1.4" fill="none"/><path d="M11 2 L11 20 M3 7 L19 15 M19 7 L3 15" stroke="currentColor" stroke-width="1" opacity="0.45"/></svg></span><span class="ph-name">prism</span></a><nav class="ph-nav"><a href="/v2/explore">Explore</a><a class="active" href="/v2/ask?q=Is%20Soroswap%20busy%3F">Ask</a><a href="/v2/home#why">Docs</a></nav><div class="ph-right"><div class="ph-net"><span class="dot"></span><span>`)
	fmt.Fprint(w, html.EscapeString(networkLabel(network)))
	fmt.Fprint(w, `</span></div></div></div></header>`)
}

func renderAskBar(w http.ResponseWriter, q string) {
	fmt.Fprint(w, `<section class="px-ask-bar"><div class="px-ask-bar-inner"><div class="px-ask-bar-eyebrow"><span class="dot"></span>Ask Prism · deterministic answers</div><form class="px-ask-input-row" action="/search/submit" method="get"><svg class="icon" width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="5"/><path d="M12 12l4 4"/></svg><input id="askInput" name="q" value="`)
	fmt.Fprint(w, html.EscapeString(q))
	fmt.Fprint(w, `" autocomplete="off"/><button class="submit" type="submit">Ask <svg width="11" height="11" viewBox="0 0 11 11" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M2 5.5h7M6 2l3.5 3.5L6 9"/></svg></button></form><div class="px-ask-examples"><span class="label">Try</span><a href="/v2/ask?q=Which%20contracts%20are%20expiring%20this%20week%3F">Which contracts are expiring this week?</a><a href="/v2/ask?q=Is%20Soroswap%20busy%3F">Is Soroswap busy?</a><a href="/v2/explore?fn=transfer&status=failed&topic=transfer">Are there failed transfers in the last hour?</a></div></div></section>`)
}

func renderAskInterpret(w http.ResponseWriter, q string, m intent.Match) {
	fmt.Fprint(w, `<section class="px-ask-interpret"><div class="px-ask-interpret-col"><div class="k">You asked</div><div class="you">`)
	fmt.Fprint(w, html.EscapeString(q))
	fmt.Fprint(w, `</div></div><div class="px-ask-interpret-col"><div class="k">Prism understood this as</div><div class="intent"><span>`)
	fmt.Fprint(w, html.EscapeString(intentTitle(m)))
	fmt.Fprint(w, `</span><div class="chips">`)
	for _, chip := range askIntentChips(m) {
		fmt.Fprintf(w, `<span class="chip%s">%s</span>`, chipClass(chip.muted), html.EscapeString(chip.label))
	}
	fmt.Fprint(w, `</div></div></div></section>`)
}

func renderAskAnswer(w http.ResponseWriter, m intent.Match, res intent.Result) {
	confidence := firstConfidence(res, m)
	fmt.Fprintf(w, `<section class="px-ask-answer %s"><div class="px-ask-answer-inner"><div class="px-ask-answer-eyebrow">Answer</div><h1 class="px-ask-answer-h1">%s</h1>`, html.EscapeString(answerTone(m, res)), askHeadlineHTML(res.Answer, m))
	fmt.Fprintf(w, `<p class="px-ask-answer-sub">%s</p>`, askSubHTML(res, m))
	fmt.Fprint(w, `<div class="px-ask-reason-eyebrow">Reason</div><div class="px-ask-reason-grid">`)
	metrics := res.Metrics
	if len(metrics) == 0 {
		metrics = []intent.Metric{{Label: "Evidence links", Value: fmt.Sprintf("%d", len(res.Evidence))}, {Label: "Confidence", Value: fmt.Sprintf("%.0f%%", confidence*100)}, {Label: "Intent", Value: string(m.ID)}}
	}
	for _, metric := range metrics {
		fmt.Fprintf(w, `<div class="px-ask-reason"><span class="k">%s</span><span class="v tnum">%s</span><span class="compare">gateway-backed deterministic input</span></div>`, html.EscapeString(metric.Label), html.EscapeString(metric.Value))
	}
	fmt.Fprint(w, `</div>`)
	fmt.Fprintf(w, `<div class="px-ask-threshold">Prism's rule: <b>%s</b>. Rule matched by <span class="mono">%s</span>. This answer was computed without an LLM.</div>`, html.EscapeString(ruleText(m)), html.EscapeString(firstNonEmptyAsk(m.Reason, string(m.ID))))
	if len(res.Evidence) > 0 {
		fmt.Fprint(w, `<div class="px-ask-reason-eyebrow">Evidence</div><div class="px-ask-evidence-grid">`)
		for _, e := range res.Evidence {
			fmt.Fprintf(w, `<a class="px-ask-evidence" href="%s"><span class="ic %s">%s</span><span class="label">%s</span><span class="meta">source</span><span class="arrow">→</span></a>`, html.EscapeString(e.Href), html.EscapeString(evidenceKind(e.Href)), html.EscapeString(evidenceInitial(e.Href)), html.EscapeString(e.Label))
		}
		fmt.Fprint(w, `</div>`)
	}
	fmt.Fprintf(w, `<div class="px-ask-foot"><div class="px-ask-confidence"><span class="k">Confidence</span><div class="meter"><span style="width: %.0f%%"></span></div><span class="val">%.0f%%</span></div><div class="px-ask-rule"><b>Deterministic rule</b> · <code>%s</code> · gateway source <code>obsrvr-gateway</code> · no LLM</div></div>`, confidence*100, confidence*100, html.EscapeString(string(m.ID)))
	fmt.Fprint(w, `</div></section>`)
}

func renderAskNext(w http.ResponseWriter, res intent.Result) {
	if len(res.Actions) == 0 {
		return
	}
	fmt.Fprint(w, `<section class="px-ask-next"><div class="px-ask-next-head">Next, you might want to</div><div class="px-ask-next-grid">`)
	for _, a := range res.Actions {
		fmt.Fprintf(w, `<a class="px-ask-next-action" href="%s"><div class="ic"><svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="7" cy="7" r="4"/><path d="M10.5 10.5L14 14"/></svg></div><div class="body"><div class="title">%s</div><div class="sub">open supporting evidence</div></div><span class="arrow">→</span></a>`, html.EscapeString(a.Href), html.EscapeString(a.Label))
	}
	fmt.Fprint(w, `</div></section>`)
}

func renderAskSide(w http.ResponseWriter, m intent.Match, res intent.Result) {
	fmt.Fprint(w, `<aside class="px-ask-side"><div class="px-ask-side-card"><h4>Supported questions</h4><ul class="px-ask-supported"><li>Is &lt;protocol&gt; busy?</li><li>Which contracts are expiring this week?</li><li>Are there failed transfers in the last hour?</li><li>Show USDC swaps today</li></ul><div class="px-ask-side-foot">Ask doesn't run an LLM. Every supported question maps to a deterministic intent handler.</div></div><div class="px-ask-side-card"><h4>This answer</h4>`)
	fmt.Fprintf(w, `<div class="px-ask-side-mini"><span class="k">Intent</span><span class="v">%s</span></div>`, html.EscapeString(string(m.ID)))
	fmt.Fprint(w, `<div class="px-ask-side-mini"><span class="k">Source</span><span class="v">gateway</span></div><div class="px-ask-side-mini"><span class="k">LLM</span><span class="v">none</span></div>`)
	fmt.Fprintf(w, `<div class="px-ask-side-mini"><span class="k">Evidence</span><span class="v">%d links</span></div>`, len(res.Evidence))
	fmt.Fprint(w, `</div></aside>`)
}

type askChip struct {
	label string
	muted bool
}

func askIntentChips(m intent.Match) []askChip {
	out := []askChip{{label: "intent · " + string(m.ID)}}
	if p := m.Slots["protocol"]; p != "" {
		out = append(out, askChip{label: "protocol · " + p})
	}
	if t := m.Slots["time"]; t != "" {
		out = append(out, askChip{label: "window · " + t, muted: true})
	}
	return out
}

func chipClass(muted bool) string {
	if muted {
		return " muted"
	}
	return ""
}
func firstConfidence(res intent.Result, m intent.Match) float64 {
	if res.Confidence > 0 {
		return res.Confidence
	}
	return m.Confidence
}
func networkLabel(n string) string {
	if n == "" {
		return "Mainnet"
	}
	return strings.ToUpper(n[:1]) + n[1:]
}

func intentTitle(m intent.Match) string {
	switch m.ID {
	case intent.ProtocolBusy:
		return "Protocol activity check"
	case intent.ExpiringContracts:
		return "Contract TTL risk check"
	default:
		return "Question intent"
	}
}
func answerTone(m intent.Match, res intent.Result) string {
	if strings.Contains(strings.ToLower(res.Answer), "busy") || m.ID == intent.ExpiringContracts {
		return "warn"
	}
	return ""
}
func ruleText(m intent.Match) string {
	if m.ID == intent.ProtocolBusy {
		return "protocol activity is busy when its computed activity score exceeds the configured threshold"
	}
	if m.ID == intent.ExpiringContracts {
		return "contracts are included when gateway TTL data flags them as needing attention"
	}
	return "the query must match a registered deterministic intent"
}
func evidenceKind(href string) string {
	if strings.Contains(href, "contract") {
		return "contract"
	}
	if strings.Contains(href, "ledger") {
		return "ledger"
	}
	return "explore"
}
func evidenceInitial(href string) string {
	if strings.Contains(href, "contract") {
		return "C"
	}
	if strings.Contains(href, "ledger") {
		return "L"
	}
	return "E"
}

func askHeadlineHTML(answer string, m intent.Match) string {
	escaped := html.EscapeString(answer)
	for _, phrase := range []string{"looks busy", "looks moderately active", "looks quiet", "near TTL expiration", "flagged as near TTL expiration"} {
		if strings.Contains(escaped, phrase) {
			cls := "verdict"
			if strings.Contains(phrase, "busy") || strings.Contains(phrase, "expiration") {
				cls += " warn"
			} else if strings.Contains(phrase, "quiet") {
				cls += " good"
			}
			return strings.Replace(escaped, phrase, `<span class="`+cls+`">`+phrase+`</span>`, 1)
		}
	}
	return escaped
}

func askSubHTML(res intent.Result, m intent.Match) string {
	if len(res.Metrics) > 0 {
		return `The reason grid below shows the deterministic inputs Prism used. The evidence chips link to raw gateway-backed data.`
	}
	if len(res.Warnings) > 0 {
		return html.EscapeString(strings.Join(res.Warnings, " "))
	}
	return `This answer is templated from a registered intent and linked evidence. It is not generated by a language model.`
}

func firstNonEmptyAsk(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
