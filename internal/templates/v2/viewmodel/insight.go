package viewmodelv2

type InsightDetailData struct {
	Network               string
	InsightID             string
	InsightLabel          string
	State                 string
	StatusLabel           string
	Tone                  string
	Title                 string
	Summary               string
	Detail                string
	SubjectLabel          string
	SubjectID             string
	SubjectHref           string
	IdentityDetail        string
	Metrics               []InsightDetailMetric
	RuleLabel             string
	RuleSummary           string
	MatchSummary          string
	ObservedWindowLabel   string
	BaselineWindowLabel   string
	ObservedLedgerRange   string
	SourceLedgerLabel     string
	Contributors          []InsightDetailContributor
	Samples               []InsightDetailSample
	Evidence              []HomeInsightEvidenceLink
	Caveats               []string
	ProvenanceSources     []string
	CompleteThroughLedger string
	UpdatedLabel          string
	ErrorTitle            string
	ErrorMessage          string
	RetryHref             string
	HomeHref              string
}

type InsightDetailMetric struct {
	Label string
	Value string
}

type InsightDetailContributor struct {
	Rank        string
	Dimension   string
	Label       string
	Key         string
	ShowKey     bool
	CountLabel  string
	ShareLabel  string
	LedgerLabel string
	Href        string
}

type InsightDetailSample struct {
	Rank          string
	KindLabel     string
	TxHash        string
	TxLabel       string
	TxHref        string
	LedgerLabel   string
	LedgerHref    string
	Context       string
	ContractID    string
	ContractLabel string
	ContractHref  string
}

func (data InsightDetailData) Available() bool {
	return data.ErrorTitle == ""
}
