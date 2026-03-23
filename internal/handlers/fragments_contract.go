package handlers

import (
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/fragments"
)

// ContractInfoFragment returns the contract info key-value section.
func (h *Handlers) ContractInfoFragment(w http.ResponseWriter, r *http.Request) {
	data := mockContractDetailData()

	if err := fragments.ContractInfo(data).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load contract info", r.URL.Path).Render(r.Context(), w)
	}
}

// ContractFunctionsFragment returns the top functions table.
func (h *Handlers) ContractFunctionsFragment(w http.ResponseWriter, r *http.Request) {
	data := mockContractDetailData()

	if err := fragments.ContractFunctions(data.Functions).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load functions", r.URL.Path).Render(r.Context(), w)
	}
}

// ContractInvocationsFragment returns the recent invocations table.
func (h *Handlers) ContractInvocationsFragment(w http.ResponseWriter, r *http.Request) {
	data := mockContractDetailData()

	if err := fragments.ContractInvocations(data.Invocations).Render(r.Context(), w); err != nil {
		h.Logger.Error("render fragment", "error", err)
		fragments.FragmentError("Could not load invocations", r.URL.Path).Render(r.Context(), w)
	}
}
