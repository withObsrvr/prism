package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// ContractInfoFragment returns the contract info key-value section.
func (h *Handlers) ContractInfoFragment(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id") // Will be used when wiring live data.
	data := mockContractDetailData()

	if err := fragments.ContractInfo(data).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load contract info", err)
	}
}

// ContractFunctionsFragment returns the top functions table.
func (h *Handlers) ContractFunctionsFragment(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id")
	data := mockContractDetailData()

	if err := fragments.ContractFunctions(data.Functions).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load functions", err)
	}
}

// ContractInvocationsFragment returns the recent invocations table.
func (h *Handlers) ContractInvocationsFragment(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id")
	data := mockContractDetailData()

	if err := fragments.ContractInvocations(data.Invocations).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not load invocations", err)
	}
}
