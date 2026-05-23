package handlers

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"

	"github.com/withObsrvr/prism/internal/templates/pages"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// TransactionCardV2 renders the Open Graph PNG image for transaction links.
func (h *Handlers) TransactionCardV2(w http.ResponseWriter, r *http.Request) {
	data, network := h.txCardData(r)
	img := renderTransactionCardPNG(data, network)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := png.Encode(w, img); err != nil {
		h.Logger.Error("encode transaction card", "hash", data.ShortHash, "error", err)
	}
}

// TransactionCardSVGV2 exposes the matching vector source for debugging the transaction card.
func (h *Handlers) TransactionCardSVGV2(w http.ResponseWriter, r *http.Request) {
	data, network := h.txCardData(r)
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(buildTransactionCardSVG(data, network)))
}

func (h *Handlers) txCardData(r *http.Request) (pages.TxReceiptData, string) {
	hash := r.PathValue("hash")
	shortHash := hash
	if len(hash) > 12 {
		shortHash = hash[:6] + "..." + hash[len(hash)-4:]
	}
	network := networkFromRequest(r)

	var data pages.TxReceiptData
	if h.useLiveData(r) {
		if live, err := h.buildTxReceiptData(r, network, hash, shortHash); err == nil {
			data = live
		} else {
			h.Logger.Warn("live transaction card data failed, falling back to mock", "hash", shortHash, "error", err)
		}
	}
	if data.Hash == "" {
		data = mockTxReceiptData(hash, shortHash)
	}
	return data, network
}

func renderTransactionCardPNG(data pages.TxReceiptData, network string) image.Image {
	const w, h = 1200, 630
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fill(img, img.Bounds(), rgb(251, 250, 247))
	drawSoftWash(img, rgb(255, 127, 80))

	ink := rgb(31, 27, 22)
	muted := rgb(86, 84, 78)
	soft := rgb(154, 150, 140)
	emerald := rgb(4, 120, 87)
	violet := rgb(94, 79, 168)
	cyan := rgb(31, 78, 95)
	amber := rgb(185, 132, 41)
	red := rgb(217, 82, 30)

	regular := mustFace(goregular.TTF, 22)
	regularLarge := mustFace(goregular.TTF, 60)
	bold := mustFace(gobold.TTF, 16)
	brand := mustFace(gobold.TTF, 36)
	mono := mustFace(goregular.TTF, 15)

	drawPrismLogo(img, 64, 50, brand, ink)
	drawRightString(img, 1136, 88, firstNonEmptyTxCard(data.Timestamp, "transaction")+" · "+strings.ToUpper(networkLabelCard(network)), mono, soft)

	status := txCardStatus(data)
	statusColor := emerald
	if strings.EqualFold(status, "FAILED") {
		statusColor = red
	}
	drawString(img, 64, 200, "TRANSACTION "+strings.ToUpper(firstNonEmptyTxCard(data.ShortHash, shortHashCard(data.Hash)))+" · "+status, bold, statusColor)

	title := txCardTitle(data)
	lines := wrapCardLine(title, 32)
	if len(lines) == 0 {
		lines = []string{"Transaction decoded"}
	}
	for i, line := range lines {
		if i > 1 {
			break
		}
		drawString(img, 64, 302+(i*66), line, regularLarge, ink)
	}

	drawString(img, 64, 432, txCardSubtitle(data), regular, muted)
	drawString(img, 64, 468, txCardMetaLine(data), regular, muted)

	x := 64
	for _, seg := range []struct {
		label string
		width int
		c     color.RGBA
	}{
		{"source", 210, violet},
		{"contract", 230, cyan},
		{"events", 170, emerald},
		{"fee", 150, amber},
		{"status", 120, statusColor},
		{"hash", 188, rgb(201, 189, 163)},
	} {
		fill(img, image.Rect(x, 540, x+seg.width, 576), seg.c)
		drawString(img, x+10, 564, seg.label, mono, rgb(251, 250, 247))
		x += seg.width + 2
	}
	drawString(img, 64, 610, "prism.withobsrvr.com/v2/tx/"+data.Hash, mono, statusColor)
	return img
}

func buildTransactionCardSVG(data pages.TxReceiptData, network string) string {
	status := txCardStatus(data)
	statusColor := "#047857"
	if strings.EqualFold(status, "FAILED") {
		statusColor = "#D9521E"
	}
	var b bytes.Buffer
	b.WriteString(`<svg viewBox="0 0 1200 630" xmlns="http://www.w3.org/2000/svg" preserveAspectRatio="xMidYMid meet">`)
	b.WriteString(`<rect width="1200" height="630" fill="#FBFAF7"/>`)
	b.WriteString(`<defs><radialGradient id="tx-warm" cx="0" cy="0" r="900" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#FF7F50" stop-opacity="0.06"/><stop offset="1" stop-color="#FF7F50" stop-opacity="0"/></radialGradient></defs><rect width="1200" height="630" fill="url(#tx-warm)"/>`)
	b.WriteString(`<g transform="translate(64, 50)"><g><rect x="0" y="0" width="18.6" height="8" fill="#1FA098"/><rect x="18.6" y="0" width="23.5" height="8" fill="#9B7BD8"/><rect x="42.1" y="0" width="10.7" height="8" fill="#E27160"/><rect x="52.8" y="0" width="12.9" height="8" fill="#A89770"/><rect x="65.7" y="0" width="6.3" height="8" fill="#3F8FBF"/></g><text x="0" y="46" fill="#1F1B16" font-family="Georgia, serif" font-size="42" font-weight="600" letter-spacing="-1">Prism</text></g>`)
	b.WriteString(`<g transform="translate(1136, 88)" text-anchor="end"><text fill="#9A968C" font-family="ui-monospace, monospace" font-size="13">` + esc(firstNonEmptyTxCard(data.Timestamp, "transaction")+" · "+strings.ToUpper(networkLabelCard(network))) + `</text></g>`)
	b.WriteString(`<text x="64" y="200" fill="` + statusColor + `" font-family="ui-monospace, monospace" font-size="16" font-weight="700" letter-spacing="0.5">TRANSACTION ` + esc(strings.ToUpper(firstNonEmptyTxCard(data.ShortHash, shortHashCard(data.Hash)))) + ` · ` + esc(status) + `</text>`)
	for i, line := range wrapCardLine(txCardTitle(data), 32) {
		if i > 1 {
			break
		}
		b.WriteString(`<text x="64" y="` + itoaCard(302+(i*66)) + `" fill="#1F1B16" font-family="Georgia, serif" font-size="60" font-weight="500" letter-spacing="-1.5">` + esc(line) + `</text>`)
	}
	b.WriteString(`<text x="64" y="432" fill="#56544E" font-family="ui-sans-serif, sans-serif" font-size="22">` + esc(txCardSubtitle(data)) + `</text>`)
	b.WriteString(`<text x="64" y="468" fill="#56544E" font-family="ui-sans-serif, sans-serif" font-size="22">` + esc(txCardMetaLine(data)) + `</text>`)
	b.WriteString(`<g transform="translate(64, 540)"><rect x="0" y="0" width="210" height="36" fill="#5E4FA8"/><rect x="212" y="0" width="230" height="36" fill="#1F4E5F"/><rect x="444" y="0" width="170" height="36" fill="#047857"/><rect x="616" y="0" width="150" height="36" fill="#B98429"/><rect x="768" y="0" width="120" height="36" fill="` + statusColor + `"/><rect x="890" y="0" width="188" height="36" fill="#C9BDA3"/></g>`)
	b.WriteString(`<text x="64" y="610" fill="` + statusColor + `" font-family="ui-monospace, monospace" font-size="15" font-weight="600">prism.withobsrvr.com/v2/tx/` + esc(data.Hash) + `</text></svg>`)
	return b.String()
}

func txCardStatus(data pages.TxReceiptData) string {
	if strings.EqualFold(data.Status, "failed") || strings.Contains(strings.ToLower(data.Status), "fail") {
		return "FAILED"
	}
	return "SUCCESSFUL"
}

func txCardTitle(data pages.TxReceiptData) string {
	for _, v := range []string{data.HeroTitle, stripHTMLCard(data.SummaryHTML), stripHTMLCard(data.AISummaryHTML), stripHTMLCard(data.HumanNarrative)} {
		v = strings.TrimSpace(v)
		if v != "" {
			return truncateRunes(v, 70)
		}
	}
	return "Transaction decoded"
}

func txCardSubtitle(data pages.TxReceiptData) string {
	parts := []string{}
	if data.Ledger != "" {
		parts = append(parts, "Ledger "+data.Ledger)
	} else if data.LedgerRaw != "" {
		parts = append(parts, "Ledger "+data.LedgerRaw)
	}
	if data.OpsCount != "" {
		parts = append(parts, data.OpsCount+" operations")
	}
	if data.EventsCount != "" {
		parts = append(parts, data.EventsCount+" events")
	}
	if len(parts) == 0 {
		return "Prism decoded this transaction from ledger evidence."
	}
	return strings.Join(parts, " · ")
}

func txCardMetaLine(data pages.TxReceiptData) string {
	parts := []string{}
	if data.ContractName != "" {
		parts = append(parts, data.ContractName)
	}
	if data.ContractFn != "" {
		parts = append(parts, data.ContractFn+"()")
	} else if data.DownstreamFunctionName != "" {
		parts = append(parts, data.DownstreamFunctionName+"()")
	}
	fee := firstNonEmptyTxCard(data.FeePaidXLM, data.FeePaid)
	if fee != "" {
		parts = append(parts, "fee "+fee)
	}
	if len(parts) == 0 {
		return "Hash " + firstNonEmptyTxCard(data.ShortHash, shortHashCard(data.Hash))
	}
	return strings.Join(parts, " · ")
}

func firstNonEmptyTxCard(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func shortHashCard(hash string) string {
	if len(hash) > 12 {
		return hash[:6] + "..." + hash[len(hash)-4:]
	}
	return hash
}

func wrapCardLine(s string, limit int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	lines := []string{}
	line := words[0]
	for _, word := range words[1:] {
		if len([]rune(line))+1+len([]rune(word)) > limit {
			lines = append(lines, line)
			line = word
			continue
		}
		line += " " + word
	}
	return append(lines, line)
}

func itoaCard(v int) string {
	if v == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
