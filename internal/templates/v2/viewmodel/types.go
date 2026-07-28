package viewmodelv2

import (
	"time"

	componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"
)

type HomeSectionState string

const (
	HomeSectionReady       HomeSectionState = "ready"
	HomeSectionPartial     HomeSectionState = "partial"
	HomeSectionStale       HomeSectionState = "stale"
	HomeSectionEmpty       HomeSectionState = "empty"
	HomeSectionUnavailable HomeSectionState = "unavailable"
)

type HomeSectionStatus struct {
	State      HomeSectionState
	Message    string
	AsOfLedger int64
	AsOfTime   time.Time
	Warnings   []string
	Retryable  bool
}

type HomeSpectrogramSegment struct {
	Kind  string
	Label string
	Count int
	Style string
}

type HomeSpectrogramColumn struct {
	Sequence             int64
	SequenceLabel        string
	Href                 string
	ClosedAt             string
	AgeLabel             string
	TransactionCount     int
	IncludedOperations   int
	SuccessfulOperations int
	FailedOperations     int
	Introducer           string
	HeightStyle          string
	FailureStyle         string
	Segments             []HomeSpectrogramSegment
	AccessibleLabel      string
	Latest               bool
}

type HomeSpectrogramLegendItem struct {
	Kind       string
	Label      string
	Count      int
	Percentage string
}

type HomeTimelineData struct {
	Status          HomeSectionStatus
	Network         string
	PollURL         string
	Freshness       string
	HeaderState     string
	HeaderLedger    string
	HeaderAge       string
	HeaderTxCount   string
	WindowLabel     string
	DetailLabel     string
	StartSequence   string
	EndSequence     string
	ColumnGridStyle string
	Columns         []HomeSpectrogramColumn
	Legend          []HomeSpectrogramLegendItem
	FailureCount    int
	FailurePercent  string
	AsOfLedgerLabel string
	DemoData        bool
}

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
	LedgerNumber             string
	TransactionCount         string
	IncludedOperationCount   string
	SuccessfulOperationCount string
	FailedOperationCount     string
	Meta                     string
	Introducer               string
	Chips                    []componentsv2.LedgerMetricChip
	InstructionsPct          int
	ReadWritePct             int
	CloseTime                string
	Age                      string
	SideMeta                 string
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
	Href     string
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
	Href       string
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
	MockMode    bool
	TimelineURL string
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
