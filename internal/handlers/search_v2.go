package handlers

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/intent"
	prismsearch "github.com/withObsrvr/prism/internal/search"
)

func (h *Handlers) SearchSuggest(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if redirect := h.detectSmartRedirect(r.Context(), networkFromRequest(r), q); redirect != "" {
		label := directSuggestionLabel(q, redirect)
		fmt.Fprintf(w, `<div class="v2-suggest-panel"><a class="v2-suggest-row primary" href="%s"><span><b>%s</b><small>Direct match, press Enter to open</small></span><strong>Open</strong></a></div>`, html.EscapeString(redirect), html.EscapeString(label))
		return
	}
	parsed, confidence := prismsearch.Parse(q)
	matches := prismsearch.DefaultRegistry().Search(q, 6)
	intentMatch, hasIntent := intent.DefaultRegistry().Match(q)
	fmt.Fprint(w, `<div class="v2-suggest-panel">`)
	if hasIntent && intentMatch.Confidence >= 0.65 {
		fmt.Fprintf(w, `<a class="v2-suggest-row primary" href="/v2/ask?q=%s"><span><b>Answer this question</b><small>%s</small></span><strong>Ask</strong></a>`, html.EscapeString(url.QueryEscape(q)), html.EscapeString(questionIntentSummary(intentMatch)))
	}
	if qs := parsed.QueryString(); confidence >= 0.45 && qs != "" {
		fmt.Fprintf(w, `<a class="v2-suggest-row primary" href="/v2/explore?%s"><span><b>Explore this activity</b><small>%s</small></span><strong>Run</strong></a>`, html.EscapeString(qs), html.EscapeString(exploreIntentSummary(parsed)))
	}
	for _, m := range matches {
		fmt.Fprintf(w, `<a class="v2-suggest-row" href="%s"><span><b>%s</b><small>%s</small></span><em>%s</em></a>`, html.EscapeString(m.Href), html.EscapeString(m.Name), html.EscapeString(m.Subtitle), html.EscapeString(string(m.Type)))
	}
	fmt.Fprintf(w, `<a class="v2-suggest-footer" href="/v2/explore?q=%s">Search recent activity for “%s”</a>`, html.EscapeString(url.QueryEscape(q)), html.EscapeString(q))
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
	if redirect := h.detectSmartRedirect(r.Context(), networkFromRequest(r), q); redirect != "" {
		hxOrRedirect(w, r, redirect)
		return
	}
	if match, ok := intent.DefaultRegistry().Match(q); ok && match.Confidence >= 0.65 {
		hxOrRedirect(w, r, "/v2/ask?q="+url.QueryEscape(q))
		return
	}
	parsed, confidence := prismsearch.Parse(q)
	mergeExploreForm(r, &parsed)
	if confidence >= 0.45 || exploreFormQuery(r) != "" {
		if qs := parsed.QueryString(); qs != "" {
			hxOrRedirect(w, r, "/v2/explore?"+qs)
			return
		}
	}
	hxOrRedirect(w, r, "/v2/explore?q="+url.QueryEscape(q))
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

func directSuggestionLabel(q, redirect string) string {
	switch {
	case strings.Contains(redirect, "/tx/"):
		return "Transaction " + gateway.ShortHash(q)
	case strings.Contains(redirect, "/account/"):
		return "Account " + gateway.ShortAddress(q)
	case strings.Contains(redirect, "/contract/"):
		return "Contract " + gateway.ShortAddress(q)
	case strings.Contains(redirect, "/ledger/"):
		return "Ledger #" + q
	default:
		return "Direct result"
	}
}

func questionIntentSummary(m intent.Match) string {
	switch m.ID {
	case intent.ExpiringContracts:
		return "Contracts near TTL expiration · " + m.Slots["time"]
	case intent.ProtocolBusy:
		return "Protocol activity · " + m.Slots["protocol"] + " · " + m.Slots["time"]
	default:
		return "Deterministic answer with evidence"
	}
}

func exploreIntentSummary(q prismsearch.Query) string {
	parts := []string{"recent activity"}
	if q.Topic != "" {
		parts = append(parts, "topic "+q.Topic)
	}
	if q.Fn != "" {
		parts = append(parts, "function "+q.Fn)
	}
	if q.Asset != "" {
		parts = append(parts, "asset "+q.Asset)
	}
	if q.Status != "" {
		parts = append(parts, q.Status)
	}
	if q.Time != "" {
		parts = append(parts, q.Time)
	}
	return strings.Join(parts, " · ")
}
