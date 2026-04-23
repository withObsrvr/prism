package viewmodelv2

import componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"

type HeroData struct {
	Eyebrow      string
	HeadlineHTML string
	Body         string
	RoleCopyJSON string
}

type PromptData struct {
	Placeholder string
}

type AlertData struct {
	Title string
	Body  string
	Meta  string
	CTA   string
}

type FilterChipData struct {
	Label    string
	DotColor string
	Active   bool
}

type LedgerRowData struct {
	LedgerNumber     string
	TransactionCount string
	Meta             string
	Chips            []componentsv2.LedgerMetricChip
	InstructionsPct  int
	ReadWritePct     int
	CloseTime        string
	Age              string
	SideMeta         string
}

type LedgerFeedData struct {
	Title   string
	Copy    string
	Note    string
	Filters []FilterChipData
	Rows    []LedgerRowData
}

type AttentionCardData struct {
	Kicker   string
	Value    string
	Tone     string
	Body     string
	BarColor string
	BarWidth string
	CTA      string
}

type AttentionSectionData struct {
	Title string
	Copy  string
	Cards []AttentionCardData
}

type LeaderCardData struct {
	Label      string
	Value      string
	Entity     string
	EntityMark string
	EntityTone string
	Body       string
}

type LeadersSectionData struct {
	Title string
	Copy  string
	Cards []LeaderCardData
}

type UtilizationCardData struct {
	Label     string
	ValueMain string
	ValueUnit string
	Tone      string
	Body      string
	BarColor  string
	BarWidth  string
}

type UtilizationSectionData struct {
	Title string
	Copy  string
	Cards []UtilizationCardData
}

type HomeData struct {
	Header      componentsv2.HeaderData
	Hero        HeroData
	Prompt      PromptData
	Alert       AlertData
	LedgerFeed  LedgerFeedData
	FeedJSON    string
	FeedLive    bool
	Attention   AttentionSectionData
	Leaders     LeadersSectionData
	Utilization UtilizationSectionData
	FooterItems []string
}
