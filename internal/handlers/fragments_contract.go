package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

// buildContractFragmentData fetches contract detail data for fragment rendering.
// Returns nil only when live data was not requested.
func (h *Handlers) buildContractFragmentData(r *http.Request, network, contractID string) *pages.ContractDetailData {
	if !h.useLiveData(r) {
		return nil
	}
	data, err := h.buildContractDetailData(r, network, contractID)
	if err != nil {
		h.Logger.Warn("live contract data failed", "error", err, "contract", contractID)
		fallback := unavailableContractDetailData(contractID, network)
		return &fallback
	}
	return &data
}

// ContractInfoFragment returns the contract info key-value section.
func (h *Handlers) ContractInfoFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	data := h.buildContractFragmentData(r, network, id)
	if data == nil {
		mock := mockContractDetailData()
		data = &mock
	}

	if err := fragments.ContractInfo(*data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load contract info", err)
	}
}

// ContractFunctionsFragment returns the top functions table.
func (h *Handlers) ContractFunctionsFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	data := h.buildContractFragmentData(r, network, id)
	if data == nil {
		mock := mockContractDetailData()
		data = &mock
	}

	if err := fragments.ContractFunctions(data.Functions).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load functions", err)
	}
}

// ContractInvocationsFragment returns the recent invocations table.
func (h *Handlers) ContractInvocationsFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	network := networkFromRequest(r)

	data := h.buildContractFragmentData(r, network, id)
	if data == nil {
		mock := mockContractDetailData()
		data = &mock
	}

	if err := fragments.ContractInvocations(data.Invocations).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load invocations", err)
	}
}
