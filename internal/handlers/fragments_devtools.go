package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// EventsStreamFragment returns the events firehose table.
func (h *Handlers) EventsStreamFragment(w http.ResponseWriter, r *http.Request) {
	data := mockEventsFirehoseData()

	if err := fragments.EventsStream(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load events", r.URL.Path).Render(r.Context(), w)
	}
}
