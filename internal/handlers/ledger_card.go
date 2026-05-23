package handlers

import (
	"bytes"
	"html"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
	"net/http"
	"strings"

	"github.com/withObsrvr/prism/internal/templates/pages"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// LedgerCardV2 renders the Open Graph PNG image used by X, Discord, Slack, and other unfurlers.
func (h *Handlers) LedgerCardV2(w http.ResponseWriter, r *http.Request) {
	data, network := h.ledgerCardData(r)
	img := renderLedgerCardPNG(data, network)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := png.Encode(w, img); err != nil {
		h.Logger.Error("encode ledger card", "error", err)
	}
}

// LedgerCardSVGV2 exposes the design source for the social card. It is useful for visual debugging;
// og:image still points at the PNG route because social crawlers handle PNG more consistently.
func (h *Handlers) LedgerCardSVGV2(w http.ResponseWriter, r *http.Request) {
	data, network := h.ledgerCardData(r)
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(buildLedgerCardSVG(data, network)))
}

func (h *Handlers) ledgerCardData(r *http.Request) (pages.LedgerDetailData, string) {
	sequence := r.PathValue("sequence")
	network := networkFromRequest(r)

	var data pages.LedgerDetailData
	if h.useLiveData(r) {
		if live, err := h.buildLedgerDetailData(r, network, sequence); err == nil {
			data = live
		} else {
			h.Logger.Warn("live ledger card data failed, falling back to mock", "error", err)
		}
	}
	if data.Sequence == "" {
		data = mockLedgerDetailData(sequence)
	}
	enrichLedgerDetailV2(&data)
	return data, network
}

func renderLedgerCardPNG(data pages.LedgerDetailData, network string) image.Image {
	const w, h = 1200, 630
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fill(img, img.Bounds(), rgb(251, 250, 247))

	// Match the old popover card composition: warm paper, subtle radial wash, large ledger story,
	// and the categorical strip at the bottom.
	drawSoftWash(img, rgb(255, 127, 80))
	ink := rgb(31, 27, 22)
	muted := rgb(86, 84, 78)
	soft := rgb(154, 150, 140)
	orange := rgb(217, 82, 30)

	regular := mustFace(goregular.TTF, 22)
	regularLarge := mustFace(goregular.TTF, 64)
	regularHuge := mustFace(goregular.TTF, 92)
	bold := mustFace(gobold.TTF, 16)
	brand := mustFace(gobold.TTF, 36)
	mono := mustFace(goregular.TTF, 15)

	drawPrismLogo(img, 64, 50, brand, ink)
	drawRightString(img, 1136, 88, data.ClosedAt+" · "+strings.ToUpper(networkLabelCard(network)), mono, soft)

	drawString(img, 64, 200, "LEDGER "+data.Sequence+" · "+strings.ToUpper(firstNonEmptyCard(data.ActivityTag, "SOROBAN-AWARE")), bold, orange)
	drawString(img, 64, 310, data.OpCount+" operations", regularHuge, ink)
	drawString(img, 64, 396, "across "+data.TxCount+" transactions", regularLarge, muted)

	drawString(img, 64, 466, truncateRunes(sharePreviewLineCard(data), 82), regular, ink)
	drawString(img, 64, 498, data.SorobanCalls+" Soroban calls · "+data.EventsEmitted+" events · "+data.TxFailed+" failed", regular, muted)

	x := 64
	for _, seg := range []struct {
		width int
		c     color.RGBA
	}{
		{302, rgb(94, 79, 168)},
		{249, rgb(31, 78, 95)},
		{211, rgb(122, 116, 106)},
		{136, rgb(185, 132, 41)},
		{98, orange},
		{76, rgb(201, 189, 163)},
	} {
		fill(img, image.Rect(x, 540, x+seg.width, 576), seg.c)
		x += seg.width + 2
	}
	drawString(img, 64, 610, "prism.withobsrvr.com/v2/ledger/"+data.SequenceRaw, mono, orange)
	return img
}

func buildLedgerCardSVG(data pages.LedgerDetailData, network string) string {
	seq := html.EscapeString(data.SequenceRaw)
	var b bytes.Buffer
	b.WriteString(`<svg viewBox="0 0 1200 630" xmlns="http://www.w3.org/2000/svg" preserveAspectRatio="xMidYMid meet">`)
	b.WriteString(`<rect width="1200" height="630" fill="#FBFAF7"/>`)
	b.WriteString(`<defs><radialGradient id="og-warm-` + seq + `" cx="0" cy="0" r="900" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#FF7F50" stop-opacity="0.06"/><stop offset="1" stop-color="#FF7F50" stop-opacity="0"/></radialGradient></defs>`)
	b.WriteString(`<rect width="1200" height="630" fill="url(#og-warm-` + seq + `)"/>`)
	b.WriteString(`<g transform="translate(64, 50)"><g><rect x="0" y="0" width="18.6" height="8" fill="#1FA098"/><rect x="18.6" y="0" width="23.5" height="8" fill="#9B7BD8"/><rect x="42.1" y="0" width="10.7" height="8" fill="#E27160"/><rect x="52.8" y="0" width="12.9" height="8" fill="#A89770"/><rect x="65.7" y="0" width="6.3" height="8" fill="#3F8FBF"/></g><text x="0" y="46" fill="#1F1B16" font-family="Georgia, serif" font-size="42" font-weight="600" letter-spacing="-1">Prism</text></g>`)
	b.WriteString(`<g transform="translate(1136, 88)" text-anchor="end"><text fill="#9A968C" font-family="ui-monospace, monospace" font-size="13">` + esc(data.ClosedAt+" · "+strings.ToUpper(networkLabelCard(network))) + `</text></g>`)
	b.WriteString(`<text x="64" y="200" fill="#D9521E" font-family="ui-monospace, monospace" font-size="16" font-weight="700" letter-spacing="0.5">LEDGER ` + esc(data.Sequence) + ` · ` + esc(strings.ToUpper(firstNonEmptyCard(data.ActivityTag, "SOROBAN-AWARE"))) + `</text>`)
	b.WriteString(`<text x="64" y="310" fill="#1F1B16" font-family="Georgia, serif" font-size="92" font-weight="500" letter-spacing="-2">` + esc(data.OpCount) + ` operations</text>`)
	b.WriteString(`<text x="64" y="396" fill="#56544E" font-family="Georgia, serif" font-size="64" font-weight="500" letter-spacing="-1">across ` + esc(data.TxCount) + ` transactions</text>`)
	b.WriteString(`<text x="64" y="466" fill="#1F1B16" font-family="ui-sans-serif, sans-serif" font-size="22">` + esc(truncateRunes(sharePreviewLineCard(data), 82)) + `</text>`)
	b.WriteString(`<text x="64" y="498" fill="#56544E" font-family="ui-sans-serif, sans-serif" font-size="22">` + esc(data.SorobanCalls+" Soroban calls · "+data.EventsEmitted+" events · "+data.TxFailed+" failed") + `</text>`)
	b.WriteString(`<g transform="translate(64, 540)"><rect x="0" y="0" width="302" height="36" fill="#5E4FA8"/><rect x="304" y="0" width="249" height="36" fill="#1F4E5F"/><rect x="555" y="0" width="211" height="36" fill="#7A746A"/><rect x="768" y="0" width="136" height="36" fill="#B98429"/><rect x="906" y="0" width="98" height="36" fill="#D9521E"/><rect x="1006" y="0" width="76" height="36" fill="#C9BDA3"/></g>`)
	b.WriteString(`<text x="64" y="610" fill="#D9521E" font-family="ui-monospace, monospace" font-size="15" font-weight="600">prism.withobsrvr.com/v2/ledger/` + esc(data.SequenceRaw) + `</text>`)
	b.WriteString(`</svg>`)
	return b.String()
}

func fill(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	imagedraw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, imagedraw.Src)
}

func drawSoftWash(img *image.RGBA, c color.RGBA) {
	for y := 0; y < 630; y++ {
		for x := 0; x < 1200; x++ {
			d := x*x + y*y
			if d > 900*900 {
				continue
			}
			a := uint8((900*900 - d) * 16 / (900 * 900))
			if a == 0 {
				continue
			}
			base := img.RGBAAt(x, y)
			img.SetRGBA(x, y, blend(base, color.RGBA{R: c.R, G: c.G, B: c.B, A: a}))
		}
	}
}

func blend(dst, src color.RGBA) color.RGBA {
	a := uint32(src.A)
	return color.RGBA{R: uint8((uint32(src.R)*a + uint32(dst.R)*(255-a)) / 255), G: uint8((uint32(src.G)*a + uint32(dst.G)*(255-a)) / 255), B: uint8((uint32(src.B)*a + uint32(dst.B)*(255-a)) / 255), A: 255}
}

func mustFace(ttf []byte, size float64) font.Face {
	f, err := opentype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic(err)
	}
	return face
}

func drawString(img *image.RGBA, x, y int, text string, face font.Face, c color.RGBA) {
	d := &font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(text)
}

func drawRightString(img *image.RGBA, x, y int, text string, face font.Face, c color.RGBA) {
	d := &font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face}
	w := d.MeasureString(text).Ceil()
	d.Dot = fixed.P(x-w, y)
	d.DrawString(text)
}

func drawPrismLogo(img *image.RGBA, x, y int, face font.Face, ink color.RGBA) {
	// Actual Prism lockup from the app chrome: segmented spectrum band over the wordmark.
	band := []struct {
		w int
		c color.RGBA
	}{
		{19, rgb(31, 160, 152)},
		{24, rgb(155, 123, 216)},
		{11, rgb(226, 113, 96)},
		{13, rgb(168, 151, 112)},
		{6, rgb(63, 143, 191)},
	}
	cx := x
	for _, seg := range band {
		fill(img, image.Rect(cx, y, cx+seg.w, y+8), seg.c)
		cx += seg.w
	}
	drawString(img, x, y+46, "Prism", face, ink)
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: 255} }
func esc(s string) string          { return html.EscapeString(s) }

func firstNonEmptyCard(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func networkLabelCard(v string) string {
	if strings.EqualFold(v, "testnet") {
		return "Testnet"
	}
	if strings.EqualFold(v, "futurenet") {
		return "Futurenet"
	}
	return "Mainnet"
}

func firstTxSummaryCard(data pages.LedgerDetailData) string {
	if len(data.Transactions) > 0 {
		return data.Transactions[0].Summary
	}
	return "No transactions loaded"
}

func stripHTMLCard(s string) string {
	out := make([]rune, 0, len(s))
	in := false
	for _, r := range s {
		if r == '<' {
			in = true
			continue
		}
		if r == '>' {
			in = false
			continue
		}
		if !in {
			out = append(out, r)
		}
	}
	return string(out)
}

func sharePreviewLineCard(data pages.LedgerDetailData) string {
	s := stripHTMLCard(firstTxSummaryCard(data))
	if strings.TrimSpace(s) == "" || s == "No transactions loaded" {
		return "Prism decoded this ledger's activity."
	}
	return s
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
