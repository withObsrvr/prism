package viewmodel

// ContractInterface is the presentation contract for a contract's current,
// declared specification. Observed invocation history is kept separate.
type ContractInterface struct {
	Available             bool
	ContractID            string
	DetectedType          string
	FunctionCount         int
	DeclaredTypeCount     int
	EventCount            int
	Functions             []ContractInterfaceFunction
	DeclaredTypes         []ContractDeclaredType
	Events                []ContractDeclaredEvent
	ObservedFunctions     []string
	ObservedCurrentCount  int
	ObservedOnlyFunctions []string
}

type ContractInterfaceFunction struct {
	Name      string
	Doc       string
	Signature string
	Inputs    []ContractNamedType
	Outputs   []string
	Observed  bool
}

type ContractNamedType struct {
	Name   string
	Type   string
	Doc    string
	Detail string
}

type ContractDeclaredType struct {
	Kind   string
	Name   string
	Doc    string
	Fields []ContractNamedType
}

type ContractDeclaredEvent struct {
	Name         string
	Doc          string
	DataFormat   string
	PrefixTopics []string
	Params       []ContractNamedType
}

type ContractArtifact struct {
	Available                  bool
	ContractID                 string
	DetectedType               string
	ExecutableType             string
	WASMHash                   string
	WASMSize                   string
	InstanceLastModifiedLedger string
	LiveUntilLedger            string
	ResolvedAtLedger           string
	CodeLastModifiedLedger     string
	ExecutableSource           string
	CodeSource                 string
	ProvenanceLabel            string
	ProtocolVersion            string
	Metadata                   []ContractMetadataItem
	HasWASM                    bool
	IsStellarAsset             bool
	DownloadHref               string
	RustHref                   string
}

type ContractMetadataItem struct {
	Key   string
	Value string
}
