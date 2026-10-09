package summary

const (
	LedgerBandVersion      = "ledger_bands_v2"
	LedgerThresholdVersion = "ledger_thresholds_v1"
)

// LedgerFacts contains evidence, not conclusions. Each measurement carries its
// own availability because a ledger can have complete fees and missing
// historical footprint counts at the same time.
type LedgerFacts struct {
	Sequence        int64    `json:"sequence"`
	ProtocolVersion int      `json:"protocol_version"`
	Evidence        Evidence `json:"evidence"`

	Transactions Int64Evidence     `json:"transactions"`
	Operations   Int64Evidence     `json:"operations"`
	Successful   Int64Evidence     `json:"successful"`
	Failed       Int64Evidence     `json:"failed"`
	FailureCause FailureCauseFacts `json:"failure_cause"`

	Capacity LedgerCapacityFacts    `json:"capacity"`
	Fees     LedgerFeeFacts         `json:"fees"`
	Activity LedgerActivityFacts    `json:"activity"`
	Mix      LedgerCompositionFacts `json:"composition"`
	Changes  LedgerChangeFacts      `json:"state_change"`
}

type FailureCauseFacts struct {
	LargestGroup Int64Evidence `json:"largest_group"`
	Distinct     Int64Evidence `json:"distinct"`
}

type ResourceFacts struct {
	Used  Int64Evidence `json:"used"`
	Limit Int64Evidence `json:"limit"`
}

type LedgerCapacityFacts struct {
	CPU              ResourceFacts `json:"cpu"`
	ReadEntries      ResourceFacts `json:"read_entries"`
	WriteEntries     ResourceFacts `json:"write_entries"`
	ReadBytes        ResourceFacts `json:"read_bytes"`
	WriteBytes       ResourceFacts `json:"write_bytes"`
	TransactionBytes ResourceFacts `json:"transaction_bytes"`
	EventReturnBytes ResourceFacts `json:"event_return_bytes"`
}

type LedgerFeeFacts struct {
	BaseFee         Int64Evidence `json:"base_fee"`
	InclusionFee    Int64Evidence `json:"inclusion_fee"`
	ResourceFee     Int64Evidence `json:"resource_fee"`
	RentFee         Int64Evidence `json:"rent_fee"`
	StorageWriteFee Int64Evidence `json:"storage_write_fee"`
	TotalCharged    Int64Evidence `json:"total_charged"`
}

type LedgerActivityFacts struct {
	TransactionBaseline Int64Evidence `json:"transaction_baseline"`
	OperationBaseline   Int64Evidence `json:"operation_baseline"`
}

type LedgerCompositionFacts struct {
	SorobanTransactions Int64Evidence `json:"soroban_transactions"`
}

type LedgerChangeFacts struct {
	Created     Int64Evidence `json:"created"`
	Updated     Int64Evidence `json:"updated"`
	Deleted     Int64Evidence `json:"deleted"`
	Archived    Int64Evidence `json:"archived"`
	Restored    Int64Evidence `json:"restored"`
	TTLExtended Int64Evidence `json:"ttl_extended"`
	Baseline    Int64Evidence `json:"baseline"`
}

type LedgerMetrics struct {
	FailureRate       RatioMetric           `json:"failure_rate"`
	LargestCauseShare RatioMetric           `json:"largest_cause_share"`
	Capacity          LedgerCapacityMetrics `json:"capacity"`
	Fees              LedgerFeeMetrics      `json:"fees"`
	Activity          LedgerActivityMetrics `json:"activity"`
	SorobanShare      RatioMetric           `json:"soroban_share"`
	StateChangeTotal  Int64Evidence         `json:"state_change_total"`
	StateChangeRatio  RatioMetric           `json:"state_change_ratio"`
}

type LedgerCapacityMetrics struct {
	CPU              RatioMetric `json:"cpu"`
	ReadEntries      RatioMetric `json:"read_entries"`
	WriteEntries     RatioMetric `json:"write_entries"`
	ReadBytes        RatioMetric `json:"read_bytes"`
	WriteBytes       RatioMetric `json:"write_bytes"`
	TransactionBytes RatioMetric `json:"transaction_bytes"`
	EventReturnBytes RatioMetric `json:"event_return_bytes"`
}

type LedgerFeeMetrics struct {
	InclusionMultiple RatioMetric `json:"inclusion_multiple"`
}

type LedgerActivityMetrics struct {
	TransactionRatio RatioMetric `json:"transaction_ratio"`
	OperationRatio   RatioMetric `json:"operation_ratio"`
}

type LedgerBands struct {
	Version          string                 `json:"version"`
	ThresholdVersion string                 `json:"threshold_version"`
	Evidence         EvidenceBand           `json:"evidence"`
	Capacity         LedgerCapacityBands    `json:"capacity"`
	Fees             LedgerFeeBands         `json:"fees"`
	Outcomes         OutcomeBand            `json:"outcomes"`
	Activity         ActivityBand           `json:"activity"`
	Composition      CompositionBand        `json:"composition"`
	StateChange      LedgerStateChangeBands `json:"state_change"`
}

type LedgerCapacityBands struct {
	Overall          PressureBand `json:"overall"`
	BindingResource  string       `json:"binding_resource,omitempty"`
	CPU              PressureBand `json:"cpu"`
	ReadEntries      PressureBand `json:"read_entries"`
	WriteEntries     PressureBand `json:"write_entries"`
	ReadBytes        PressureBand `json:"read_bytes"`
	WriteBytes       PressureBand `json:"write_bytes"`
	TransactionBytes PressureBand `json:"transaction_bytes"`
	EventReturnBytes PressureBand `json:"event_return_bytes"`
}

type LedgerFeeBands struct {
	Overall      FeeBand      `json:"overall"`
	Inclusion    FeeBand      `json:"inclusion"`
	Resource     Availability `json:"resource"`
	Rent         Availability `json:"rent"`
	StorageWrite Availability `json:"storage_write"`
}

type LedgerStateChangeBands struct {
	Volume   ChangeBand   `json:"volume"`
	Created  Availability `json:"created"`
	Updated  Availability `json:"updated"`
	Deleted  Availability `json:"deleted"`
	Archived Availability `json:"archived"`
	Restored Availability `json:"restored"`
	TTL      Availability `json:"ttl_extended"`
}

type LedgerSummaryState struct {
	Facts   LedgerFacts   `json:"facts"`
	Metrics LedgerMetrics `json:"metrics"`
	Bands   LedgerBands   `json:"bands"`
}
