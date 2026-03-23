package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// EventsStreamFragment returns the events firehose table.
func (h *Handlers) EventsStreamFragment(w http.ResponseWriter, r *http.Request) {
	_ = networkFromRequest(r) // Will be used when wiring live data.
	data := mockEventsFirehoseData()

	if err := fragments.EventsStream(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load events", err)
	}
}
