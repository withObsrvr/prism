package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// EventsStreamFragment returns the events firehose table.
func (h *Handlers) EventsStreamFragment(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)

	var data pages.EventsFirehoseData

	if h.useLiveData(r) {
		if live, err := h.buildEventsFirehoseData(r, network); err == nil {
			data = live
		} else {
			h.Logger.Warn("live events data failed, falling back to mock", "error", err)
		}
	}

	if data.Events == nil {
		data = mockEventsFirehoseData()
	}

	if err := fragments.EventsStream(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load events", err)
	}
}
