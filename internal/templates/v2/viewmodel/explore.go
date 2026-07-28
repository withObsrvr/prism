package viewmodelv2

// ExploreData is the v2 explore page view model.
type ExploreData struct {
	Header       ExploreHeaderData
	Filters      ExploreFilters
	Summary      ExploreSummary
	Presets      []ExplorePreset
	Rows         []ExploreRow
	SourceLive   bool
	ErrorMessage string
	HasMore      bool
	NextCursor   string
	NextHref     string
	LiveHref     string
	Loading      bool
	DemoData     bool
}

type ExploreHeaderData struct {
	Network      string
	LedgerNumber string
	AgeLabel     string
	Status       string
}

type ExploreFilters struct {
	Scope  string
	Topic  string
	Fn     string
	Asset  string
	Time   string
	Status string
	Q      string
}

type ExploreSummary struct {
	SentenceHTML  string
	MatchedLabel  string
	WindowLabel   string
	LedgerRange   string
	EventsPerSec  string
	EvidenceLabel string
}

type ExplorePreset struct {
	Name string
	Body string
	Href string
}

type ExploreRow struct {
	When         string
	Age          string
	Scope        string
	Protocol     string
	Topic        string
	Function     string
	Headline     string
	From         string
	Contract     string
	TxHash       string
	Ledger       string
	Events       string
	Fee          string
	Status       string
	StatusTone   string
	Asset        string
	EvidenceHref string
	ContractHref string
	LedgerHref   string
}
