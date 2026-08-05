package viewmodelv2

// Ledger detail v3 view model.
//
// This is a prototype surface. Its purpose is to pin down the data contract for
// the "Prism Ledger Detail v3" design before any of it is wired to live data,
// so every rendered value carries a Provenance describing which obsrvr-lake
// source would feed it and whether that source exists today.
//
// Fields whose Provenance is unavailable are still rendered with mock values —
// the page shows the finished design and flags the cost of shipping it.

// ProvenanceKind classifies how a value is obtained.
type ProvenanceKind string

const (
	// ProvenanceServed: a serving projection or query-api endpoint returns this
	// value directly.
	ProvenanceServed ProvenanceKind = "served"
	// ProvenanceDerived: computable in the handler from served fields, no new
	// projection required.
	ProvenanceDerived ProvenanceKind = "derived"
	// ProvenanceGap: no source at bronze, silver, or serving. Requires new
	// projection work before this value can be truthful.
	ProvenanceGap ProvenanceKind = "gap"
	// ProvenanceNone: not observable from chain data at all. The page says so
	// rather than estimating.
	ProvenanceNone ProvenanceKind = "none"
)

// Provenance records where a rendered value comes from.
type Provenance struct {
	Kind ProvenanceKind
	// Origin names the concrete source, e.g.
	// "serving.sv_ledger_stats_recent.close_time_seconds" or
	// "GET /api/v1/silver/ledgers/{seq}/soroban".
	Origin string
	// Note explains a derivation or, for gaps, what is missing.
	Note string
}

// IsGap reports whether this value cannot be served truthfully today.
func (p Provenance) IsGap() bool { return p.Kind == ProvenanceGap }

// Label renders the short confidence chip shown beside a value.
func (p Provenance) Label() string {
	switch p.Kind {
	case ProvenanceServed:
		return "Read"
	case ProvenanceDerived:
		return "Derived"
	case ProvenanceGap:
		return "No source"
	default:
		return "None"
	}
}

// LedgerDetailV3Data is the whole page.
type LedgerDetailV3Data struct {
	Network string

	Header   LedgerV3Header
	Lede     []string // paragraphs, pre-rendered HTML (bold, citations, terms)
	Standing []LedgerV3Standing
	Strip    LedgerV3Strip
	Capacity LedgerV3Capacity
	Fees     LedgerV3Fees
	Failures LedgerV3Failures
	Changes  LedgerV3Changes
	Chain    LedgerV3Chain
	Notes    []LedgerV3Note
	Rail     LedgerV3Rail

	TxPane    LedgerV3TxPane
	StatePane LedgerV3StatePane

	// Gaps summarises every ProvenanceGap on the page, for the banner and the
	// developer-facing summary at the foot.
	Gaps []LedgerV3Gap
}

// LedgerV3Gap is one unservable requirement.
type LedgerV3Gap struct {
	Section  string
	Field    string
	Missing  string
	Proposal string
}

type LedgerV3Header struct {
	Sequence        string // display-formatted, e.g. "55,812,406"
	SequenceRaw     string
	Hash            string
	ClosedAt        string
	ClosedRelative  string
	CloseTime       string
	ProtocolVersion string
	PrevSequence    string
	PrevSequenceRaw string

	Kicker string // "118 transactions · closed in 5.2s"

	// Headline is the interpretive title. Lead/Emphasis/Trail split so the
	// emphasised span can be styled without embedding markup.
	HeadlineLead     string
	HeadlineEmphasis string
	HeadlineTrail    string

	Badges []LedgerV3Badge

	TxTabCount    string
	StateTabCount string

	// HeadlineSource is the provenance of the claim the headline makes. For v3
	// the headline asserts a binding resource limit, so this tracks the
	// capacity provenance.
	HeadlineSource Provenance
}

type LedgerV3Badge struct {
	Label string
	Tone  string // "", "ok", "warn"
}

// LedgerV3Standing is one verdict chip in the row beneath the lede.
type LedgerV3Standing struct {
	Key    string
	Value  string
	Detail string
	Dot    string // "g", "a", "r", "v", "none"
	Source Provenance
}

type LedgerV3Strip struct {
	Ticks  []LedgerV3Tick
	Foot   []LedgerV3Stat
	Legend []LedgerV3LegendEntry
	Note   string
	Source Provenance
}

// LedgerV3Tick is one transaction in the apply-order strip.
type LedgerV3Tick struct {
	Position   int
	Kind       string // calls|payments|markets|deployments|other|fail
	HeightPct  int    // scaled from operation count
	Failed     bool
	OpCount    int
	TipTitle   string
	TipDetail  string
	TipStatus  string
	Href       string
	AriaLabel  string
	ErrorCode  string
	ProtocolID string
}

type LedgerV3Stat struct {
	Label  string
	Value  string
	Source Provenance
}

type LedgerV3LegendEntry struct {
	Label string
	Count string
	Color string // css var name, e.g. "--ph-violet"
}

type LedgerV3Capacity struct {
	Meters []LedgerV3Meter
	Note   string
}

// LedgerV3Meter is one of the four independent ledger limits.
type LedgerV3Meter struct {
	Name    string
	Pct     int
	Used    string
	Cap     string
	Note    string
	Binding bool
	Source  Provenance
}

type LedgerV3Fees struct {
	ClearingLabel   string
	ClearingValue   string
	ClearingUnit    string
	BaseFee         string
	Multiple        string
	TotalCollected  string
	HighestBidNote  string
	History         []LedgerV3FeeBar
	HistoryFromNote string
	HistoryToNote   string
	CannotTitle     string
	CannotBody      string
	Source          Provenance
	ExcludedSource  Provenance
}

type LedgerV3FeeBar struct {
	HeightPct int
	Active    bool
	Title     string
}

type LedgerV3Failures struct {
	Groups []LedgerV3FailGroup
	Aside  string
	Intro  string
	Note   string
	Source Provenance
}

type LedgerV3FailGroup struct {
	Count      int
	Title      string // pre-rendered HTML: <em> marks the interpretive clause
	Detail     string
	Code       string
	FeeNote    string
	ProtocolID string
}

type LedgerV3Changes struct {
	Total      string
	Cells      []LedgerV3ChangeCell
	EntryTypes []LedgerV3EntryType
	Note       string
	Source     Provenance
}

type LedgerV3ChangeCell struct {
	Label  string
	Count  string
	Note   string
	Tone   string // "", "warn"
	Source Provenance
}

type LedgerV3EntryType struct {
	Label string
	Count string
}

type LedgerV3Chain struct {
	Neighbors []LedgerV3Neighbor
	Note      string
	Source    Provenance
}

type LedgerV3Neighbor struct {
	Sequence string
	Note     string // pre-rendered HTML
	TxCount  string
	WritePct int
	Self     bool
	Href     string
}

type LedgerV3Note struct {
	ID     string
	Index  int
	Text   string // pre-rendered HTML
	Source string
	Href   string
	IsGap  bool
}

type LedgerV3Rail struct {
	Title    string
	Subtitle string
	Groups   []LedgerV3RailGroup
	TOC      []LedgerV3TOCEntry
	Footnote string
}

type LedgerV3RailGroup struct {
	Heading string
	Rows    []LedgerV3RailRow
}

type LedgerV3RailRow struct {
	Label string
	Value string
	Mono  bool
	Href  string
	IsGap bool
}

type LedgerV3TOCEntry struct {
	Label string
	Href  string
}

// ---- transactions pane ----

type LedgerV3TxPane struct {
	Title        string
	Intro        string
	Facets       []LedgerV3Facet
	SaidLead     string
	Distribution LedgerV3Distribution
	Rows         []LedgerV3Row
	ShownLabel   string
	TotalLabel   string
	SortOptions  []string
}

type LedgerV3Facet struct {
	Key     string
	Label   string
	PopHead string
	Options []LedgerV3FacetOption
	Source  Provenance
}

type LedgerV3FacetOption struct {
	Value string
	Label string
	Count string
}

type LedgerV3Distribution struct {
	Heading string
	Aside   string
	Bars    []LedgerV3DistBar
	AxisMin string
	AxisMid string
	AxisMax string
	Legend  string
}

type LedgerV3DistBar struct {
	HeightPct int
	Title     string
}

// LedgerV3Row is a row in either workbench pane.
type LedgerV3Row struct {
	Kind    string // facet value for data-kind / data-ct
	Result  string // ok|fail, facet value for data-res
	Stamp   string
	Say     string // pre-rendered HTML
	Meta    string // pre-rendered HTML
	Who     string
	Status  string
	Failed  bool
	Href    string
	IsGap   bool
	GapNote string
}

// ---- state changes pane ----

type LedgerV3StatePane struct {
	Title      string
	Intro      string
	Facets     []LedgerV3Facet
	SaidLead   string
	Cells      []LedgerV3ChangeCell
	Rows       []LedgerV3Row
	ShownLabel string
	TotalLabel string
	SortOption []string
}
