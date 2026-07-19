package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	pagesv2 "github.com/withObsrvr/prism/internal/templates/v2/pages"
	ui "github.com/withObsrvr/prism/internal/viewmodel"
)

const contractArtifactTimeout = 8 * time.Second

// ContractInterfaceFragment loads the declared contract specification without
// delaying the contract detail shell.
func (h *Handlers) ContractInterfaceFragment(w http.ResponseWriter, r *http.Request) {
	contractID := r.PathValue("id")
	network := networkFromRequest(r)
	model := unavailableContractInterface(contractID)

	if r.URL.Query().Get("mock") == "true" {
		model, _ = contractArtifactViews(mockContractInterfaceResponse(contractID, network), contractID, network)
	} else if h.Gateway != nil {
		ctx, cancel := context.WithTimeout(r.Context(), contractArtifactTimeout)
		response, err := h.Gateway.GetContractInterface(ctx, network, contractID)
		cancel()
		if err == nil {
			model, _ = contractArtifactViews(response, contractID, network)
		} else if h.Logger != nil {
			h.Logger.Warn("contract interface unavailable", "contract", contractID, "network", network, "error", err)
		}
	}

	if err := pagesv2.ContractInterfaceFragment(model).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not render contract interface", err)
	}
}

// ContractArtifactFragment loads current executable identity and provenance on
// first use of the WASM tab.
func (h *Handlers) ContractArtifactFragment(w http.ResponseWriter, r *http.Request) {
	contractID := r.PathValue("id")
	network := networkFromRequest(r)
	model := unavailableContractArtifact(contractID, network)

	if r.URL.Query().Get("mock") == "true" {
		_, model = contractArtifactViews(mockContractInterfaceResponse(contractID, network), contractID, network)
	} else if h.Gateway != nil {
		ctx, cancel := context.WithTimeout(r.Context(), contractArtifactTimeout)
		response, err := h.Gateway.GetContractInterface(ctx, network, contractID)
		cancel()
		if err == nil {
			_, model = contractArtifactViews(response, contractID, network)
		} else if h.Logger != nil {
			h.Logger.Warn("contract artifact unavailable", "contract", contractID, "network", network, "error", err)
		}
	}

	if err := pagesv2.ContractArtifactFragment(model).Render(r.Context(), w); err != nil {
		h.renderFragmentError(w, r, "Could not render contract artifact", err)
	}
}

// ContractInterfaceRust proxies the authenticated Gateway text representation.
func (h *Handlers) ContractInterfaceRust(w http.ResponseWriter, r *http.Request) {
	if h.Gateway == nil {
		http.Error(w, "Contract interface unavailable", http.StatusServiceUnavailable)
		return
	}
	contractID := r.PathValue("id")
	network := networkFromRequest(r)
	ctx, cancel := context.WithTimeout(r.Context(), contractArtifactTimeout)
	result, err := h.Gateway.GetContractInterfaceRust(ctx, network, contractID)
	cancel()
	if err != nil {
		writeContractArtifactError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s-interface.rs"`, contractID))
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if result.WASMHash != "" {
		w.Header().Set("X-Wasm-SHA256", result.WASMHash)
	}
	_, _ = w.Write([]byte(result.Text))
}

// ContractWASMDownload proxies the authenticated Gateway download while
// preserving validation and conditional-request headers.
func (h *Handlers) ContractWASMDownload(w http.ResponseWriter, r *http.Request) {
	if h.Gateway == nil {
		http.Error(w, "Contract WASM unavailable", http.StatusServiceUnavailable)
		return
	}
	contractID := r.PathValue("id")
	network := networkFromRequest(r)
	ctx, cancel := context.WithTimeout(r.Context(), contractArtifactTimeout)
	result, err := h.Gateway.GetContractWASM(ctx, network, contractID, r.Header.Get("If-None-Match"))
	cancel()
	if err != nil {
		writeContractArtifactError(w, err)
		return
	}
	copyHeaderIfPresent(w.Header(), "ETag", result.ETag)
	if result.StatusCode == http.StatusNotModified {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	copyHeaderIfPresent(w.Header(), "Content-Type", result.ContentType)
	copyHeaderIfPresent(w.Header(), "Content-Disposition", result.ContentDisposition)
	copyHeaderIfPresent(w.Header(), "X-Contract-ID", result.ContractID)
	copyHeaderIfPresent(w.Header(), "X-Wasm-SHA256", result.WASMHash)
	copyHeaderIfPresent(w.Header(), "X-Resolved-At-Ledger", result.ResolvedAtLedger)
	// The contract-ID route identifies the active executable, which can change
	// after an upgrade. Browsers may store the response but must revalidate it.
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(result.StatusCode)
	_, _ = w.Write(result.Body)
}

func copyHeaderIfPresent(header http.Header, key, value string) {
	if strings.TrimSpace(value) != "" {
		header.Set(key, value)
	}
}

func writeContractArtifactError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	message := "Contract artifact unavailable"
	if apiErr, ok := err.(*gateway.APIError); ok {
		switch apiErr.StatusCode {
		case http.StatusBadRequest, http.StatusNotFound, http.StatusServiceUnavailable:
			status = apiErr.StatusCode
		}
	}
	http.Error(w, message, status)
}

func contractArtifactViews(response *gateway.ContractInterface, contractID, network string) (ui.ContractInterface, ui.ContractArtifact) {
	if response == nil {
		return unavailableContractInterface(contractID), unavailableContractArtifact(contractID, network)
	}
	if response.ContractID != "" {
		contractID = response.ContractID
	}
	declaredNames := make(map[string]struct{}, len(response.Interface.Functions))
	observedNames := make(map[string]struct{}, len(response.ObservedFunctions))
	for _, name := range response.ObservedFunctions {
		observedNames[name] = struct{}{}
	}

	interfaceModel := ui.ContractInterface{
		Available:         true,
		ContractID:        contractID,
		DetectedType:      humanContractToken(response.DetectedType),
		FunctionCount:     len(response.Interface.Functions),
		DeclaredTypeCount: len(response.Interface.Structs) + len(response.Interface.Unions) + len(response.Interface.Enums) + len(response.Interface.Errors),
		EventCount:        len(response.Interface.Events),
		ObservedFunctions: append([]string(nil), response.ObservedFunctions...),
	}
	for _, fn := range response.Interface.Functions {
		declaredNames[fn.Name] = struct{}{}
		_, observed := observedNames[fn.Name]
		inputs := make([]ui.ContractNamedType, 0, len(fn.Inputs))
		for _, input := range fn.Inputs {
			inputs = append(inputs, ui.ContractNamedType{Name: input.Name, Type: input.Type, Doc: input.Doc})
		}
		interfaceModel.Functions = append(interfaceModel.Functions, ui.ContractInterfaceFunction{
			Name: fn.Name, Doc: fn.Doc, Signature: contractFunctionSignature(fn), Inputs: inputs,
			Outputs: append([]string(nil), fn.Outputs...), Observed: observed,
		})
		if observed {
			interfaceModel.ObservedCurrentCount++
		}
	}
	for _, name := range response.ObservedFunctions {
		if _, declared := declaredNames[name]; !declared {
			interfaceModel.ObservedOnlyFunctions = append(interfaceModel.ObservedOnlyFunctions, name)
		}
	}
	for _, item := range response.Interface.Structs {
		fields := make([]ui.ContractNamedType, 0, len(item.Fields))
		for _, field := range item.Fields {
			fields = append(fields, ui.ContractNamedType{Name: field.Name, Type: field.Type, Doc: field.Doc})
		}
		interfaceModel.DeclaredTypes = append(interfaceModel.DeclaredTypes, ui.ContractDeclaredType{Kind: "Struct", Name: item.Name, Doc: item.Doc, Fields: fields})
	}
	for _, item := range response.Interface.Unions {
		fields := make([]ui.ContractNamedType, 0, len(item.Cases))
		for _, unionCase := range item.Cases {
			fields = append(fields, ui.ContractNamedType{Name: unionCase.Name, Type: strings.Join(unionCase.Values, ", "), Doc: unionCase.Doc})
		}
		interfaceModel.DeclaredTypes = append(interfaceModel.DeclaredTypes, ui.ContractDeclaredType{Kind: "Union", Name: item.Name, Doc: item.Doc, Fields: fields})
	}
	appendEnums := func(kind string, enums []gateway.ContractSpecEnum) {
		for _, item := range enums {
			fields := make([]ui.ContractNamedType, 0, len(item.Cases))
			for _, enumCase := range item.Cases {
				fields = append(fields, ui.ContractNamedType{Name: enumCase.Name, Type: fmt.Sprintf("%d", enumCase.Value), Doc: enumCase.Doc})
			}
			interfaceModel.DeclaredTypes = append(interfaceModel.DeclaredTypes, ui.ContractDeclaredType{Kind: kind, Name: item.Name, Doc: item.Doc, Fields: fields})
		}
	}
	appendEnums("Enum", response.Interface.Enums)
	appendEnums("Error", response.Interface.Errors)
	for _, event := range response.Interface.Events {
		params := make([]ui.ContractNamedType, 0, len(event.Params))
		for _, param := range event.Params {
			params = append(params, ui.ContractNamedType{Name: param.Name, Type: param.Type, Doc: param.Doc, Detail: humanContractToken(param.Location)})
		}
		interfaceModel.Events = append(interfaceModel.Events, ui.ContractDeclaredEvent{
			Name: event.Name, Doc: event.Doc, DataFormat: event.DataFormat,
			PrefixTopics: append([]string(nil), event.PrefixTopics...), Params: params,
		})
	}

	executable := response.Executable
	isStellarAsset := strings.EqualFold(executable.Type, "stellar_asset")
	artifactModel := ui.ContractArtifact{
		Available:                  true,
		ContractID:                 contractID,
		DetectedType:               humanContractToken(response.DetectedType),
		ExecutableType:             humanContractToken(executable.Type),
		WASMHash:                   executable.WASMHash,
		WASMSize:                   formatArtifactBytes(executable.WASMSizeBytes),
		InstanceLastModifiedLedger: formatArtifactLedger(executable.InstanceLastModifiedLedger),
		LiveUntilLedger:            formatOptionalArtifactLedger(executable.LiveUntilLedger),
		ResolvedAtLedger:           formatArtifactLedger(executable.ResolvedAtLedger),
		CodeLastModifiedLedger:     formatArtifactLedger(response.Provenance.CodeLedger),
		ExecutableSource:           humanContractToken(response.Provenance.ExecutableSource),
		CodeSource:                 humanContractToken(response.Provenance.CodeSource),
		ProvenanceLabel:            contractProvenanceLabel(response.Provenance.CodeSource, isStellarAsset),
		HasWASM:                    !isStellarAsset && executable.WASMHash != "",
		IsStellarAsset:             isStellarAsset,
		DownloadHref:               contractArtifactHref(contractID, network, "wasm"),
		RustHref:                   contractArtifactHref(contractID, network, "interface.rust"),
	}
	if version := response.Environment.InterfaceVersion; version != nil {
		artifactModel.ProtocolVersion = fmt.Sprintf("Protocol %d", version.Protocol)
		if version.PreRelease > 0 {
			artifactModel.ProtocolVersion += fmt.Sprintf(" pre-release %d", version.PreRelease)
		}
	}
	for _, item := range response.Metadata {
		artifactModel.Metadata = append(artifactModel.Metadata, ui.ContractMetadataItem{Key: item.Key, Value: item.Value})
	}
	sortContractMetadata(artifactModel.Metadata)
	return interfaceModel, artifactModel
}

func contractFunctionSignature(fn gateway.ContractSpecFunction) string {
	args := make([]string, 0, len(fn.Inputs))
	for _, input := range fn.Inputs {
		args = append(args, input.Name+": "+input.Type)
	}
	signature := "fn " + fn.Name + "(" + strings.Join(args, ", ") + ")"
	switch len(fn.Outputs) {
	case 0:
		return signature
	case 1:
		return signature + " -> " + fn.Outputs[0]
	default:
		return signature + " -> (" + strings.Join(fn.Outputs, ", ") + ")"
	}
}

func unavailableContractInterface(contractID string) ui.ContractInterface {
	return ui.ContractInterface{ContractID: contractID}
}

func unavailableContractArtifact(contractID, network string) ui.ContractArtifact {
	return ui.ContractArtifact{ContractID: contractID, DownloadHref: contractArtifactHref(contractID, network, "wasm"), RustHref: contractArtifactHref(contractID, network, "interface.rust")}
}

func formatArtifactBytes(n int64) string {
	if n <= 0 {
		return "unknown"
	}
	return fmt.Sprintf("%s (%s bytes)", formatBytes(n), gateway.FormatNumber(n))
}

func formatArtifactLedger(n int64) string {
	if n <= 0 {
		return "unknown"
	}
	return gateway.FormatNumber(n)
}

func formatOptionalArtifactLedger(n *int64) string {
	if n == nil {
		return "unknown"
	}
	return formatArtifactLedger(*n)
}

func humanContractToken(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	if value == "" {
		return "unknown"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func contractProvenanceLabel(codeSource string, stellarAsset bool) string {
	if stellarAsset || strings.EqualFold(codeSource, "protocol_builtin") {
		return "Canonical protocol interface"
	}
	switch strings.ToLower(strings.TrimSpace(codeSource)) {
	case "file_cache":
		return "Hash verified cached artifact"
	case "stellar_rpc":
		return "Hash verified from Stellar RPC"
	default:
		return "Hash verified active artifact"
	}
}

func contractArtifactHref(contractID, network, resource string) string {
	path := "/v2/contract/" + url.PathEscape(contractID) + "/" + resource
	if network != "" {
		path += "?network=" + url.QueryEscape(network)
	}
	return path
}

func mockContractInterfaceResponse(contractID, network string) *gateway.ContractInterface {
	liveUntil := int64(3794457)
	return &gateway.ContractInterface{
		ContractID: contractID,
		Network:    network,
		Executable: gateway.ContractExecutable{Type: "wasm", WASMHash: strings.Repeat("a", 64), WASMSizeBytes: 10027, InstanceLastModifiedLedger: 3673498, LiveUntilLedger: &liveUntil, ResolvedAtLedger: 3693849},
		Interface: gateway.ContractDeclaredInterface{
			Functions: []gateway.ContractSpecFunction{
				{Name: "get_price", Doc: "Read the latest price for an asset.", Inputs: []gateway.ContractSpecField{{Name: "asset", Type: "BytesN<8>"}}, Outputs: []string{"Option<PriceEntry>"}},
				{Name: "set_publishers", Doc: "Replace the publisher key set.", Inputs: []gateway.ContractSpecField{{Name: "publishers", Type: "Vec<BytesN<32>>"}}, Outputs: []string{"Result<Void, Error>"}},
			},
			Structs: []gateway.ContractSpecStruct{{Name: "PriceEntry", Fields: []gateway.ContractSpecField{{Name: "price", Type: "i128"}, {Name: "timestamp", Type: "u64"}}}},
			Errors:  []gateway.ContractSpecEnum{{Name: "Error", Cases: []gateway.ContractSpecEnumCase{{Name: "NotInitialized", Value: 2}}}},
		},
		Metadata:          []gateway.ContractInterfaceMetadata{{Key: "rsver", Value: "1.94.1"}, {Key: "rssdkver", Value: "26.0.0"}},
		Environment:       gateway.ContractInterfaceEnvironment{InterfaceVersion: &gateway.ContractInterfaceVersion{Protocol: 26}},
		Provenance:        gateway.ContractArtifactProvenance{ExecutableSource: "stellar_rpc", CodeSource: "file_cache", CodeLedger: 3673498, ResolvedAtLedger: 3693849},
		ObservedFunctions: []string{"get_price"},
	}
}

func sortContractMetadata(items []ui.ContractMetadataItem) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].Key < items[j].Key })
}
