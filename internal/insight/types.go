package insight

type Metric struct {
	Label string
	Value string
}

type EvidenceLink struct {
	Label string
	Href  string
}

type Subject struct {
	Label          string
	ID             string
	Href           string
	IdentityDetail string
}

type Provenance struct {
	Sources               []string
	CompleteThroughLedger int64
	UpdatedAt             string
}

type Interpretation struct {
	RuleID          string
	RuleVersion     string
	Title           string
	Summary         string
	Detail          string
	Severity        string
	Status          string
	WindowLabel     string
	ComparisonLabel string
	Subject         Subject
	EvidenceCount   int64
	Metrics         []Metric
	Evidence        []EvidenceLink
	Caveats         []string
	Provenance      Provenance
	Generic         bool
}
