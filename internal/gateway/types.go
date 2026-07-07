package gateway

import "encoding/json"

// NetworkStats matches the /silver/stats/network response.
type NetworkStats struct {
	GeneratedAt     string              `json:"generated_at"`
	DataFreshness   string              `json:"data_freshness"`
	Accounts        AccountStats        `json:"accounts"`
	Ledger          LedgerStats         `json:"ledger"`
	Operations24H   OperationStats      `json:"operations_24h"`
	Transactions24H TransactionStats24H `json:"transactions_24h"`
	Fees24H         FeeStats24H         `json:"fees_24h"`
	Soroban         SorobanNetStats     `json:"soroban"`
}

type TransactionStats24H struct {
	Total       int64   `json:"total"`
	Failed      int64   `json:"failed"`
	FailureRate float64 `json:"failure_rate"`
}

type FeeStats24H struct {
	MedianStroops     int64 `json:"median_stroops"`
	P99Stroops        int64 `json:"p99_stroops"`
	DailyTotalStroops int64 `json:"daily_total_stroops"`
	SurgeActive       bool  `json:"surge_active"`
}

type SorobanNetStats struct {
	ActiveContracts24H int64 `json:"active_contracts_24h"`
	AvgCPUInsns        int64 `json:"avg_cpu_insns"`
}

type AccountStats struct {
	Total      int64 `json:"total"`
	Active24H  int64 `json:"active_24h"`
	Created24H int64 `json:"created_24h"`
}

type LedgerStats struct {
	CurrentSequence     int64   `json:"current_sequence"`
	AvgCloseTimeSeconds float64 `json:"avg_close_time_seconds"`
	ProtocolVersion     int     `json:"protocol_version"`
}

type OperationStats struct {
	Total          int64 `json:"total"`
	Payments       int64 `json:"payments"`
	CreateAccount  int64 `json:"create_account"`
	AccountMerge   int64 `json:"account_merge"`
	ChangeTrust    int64 `json:"change_trust"`
	ManageOffer    int64 `json:"manage_offer"`
	ContractInvoke int64 `json:"contract_invoke"`
	Other          int64 `json:"other"`
}

// BronzeNetworkStats matches the /bronze/stats/network response.
type BronzeNetworkStats struct {
	GeneratedAt     string            `json:"generated_at"`
	DataFreshness   string            `json:"data_freshness"`
	Ledger          BronzeLedgerStats `json:"ledger"`
	Transactions24H BronzeTxStats24H  `json:"transactions_24h"`
	Operations24H   BronzeOpStats24H  `json:"operations_24h"`
}

type BronzeLedgerStats struct {
	LatestSequence      int64   `json:"latest_sequence"`
	LatestHash          string  `json:"latest_hash"`
	ClosedAt            string  `json:"closed_at"`
	ProtocolVersion     int     `json:"protocol_version"`
	AvgCloseTimeSeconds float64 `json:"avg_close_time_seconds"`
}

type BronzeTxStats24H struct {
	Total            int64 `json:"total"`
	Successful       int64 `json:"successful"`
	Failed           int64 `json:"failed"`
	SorobanCount     int64 `json:"soroban_count"`
	TotalFeesCharged int64 `json:"total_fees_charged"`
}

type BronzeOpStats24H struct {
	Total          int64 `json:"total"`
	SorobanOpCount int64 `json:"soroban_op_count"`
}

// Ledger matches the /bronze/ledgers response items.
type Ledger struct {
	Sequence            int64  `json:"sequence"`
	LedgerHash          string `json:"ledger_hash"`
	PreviousLedgerHash  string `json:"previous_ledger_hash"`
	ClosedAt            string `json:"closed_at"`
	SuccessfulTxCount   int    `json:"successful_tx_count"`
	FailedTxCount       int    `json:"failed_tx_count"`
	OperationCount      int    `json:"operation_count"`
	TxSetOperationCount int    `json:"tx_set_operation_count"`
	TransactionCount    int    `json:"transaction_count"`
	ProtocolVersion     int    `json:"protocol_version"`
	TotalCoins          int64  `json:"total_coins"`
	BaseFee             int64  `json:"base_fee"`
	BaseReserve         int64  `json:"base_reserve"`
	MaxTxSetSize        int    `json:"max_tx_set_size"`
	FeePool             int64  `json:"fee_pool"`
	SorobanFeeWrite1KB  int64  `json:"soroban_fee_write_1kb"`
	SorobanOpCount      *int   `json:"soroban_op_count"`
	TotalFeeCharged     *int64 `json:"total_fee_charged"`
	ContractEventsCount *int   `json:"contract_events_count"`
}

// LedgersResponse is the envelope for /bronze/ledgers.
type LedgersResponse struct {
	Count   int      `json:"count"`
	Start   int64    `json:"start"`
	End     int64    `json:"end"`
	Sort    string   `json:"sort"`
	Ledgers []Ledger `json:"ledgers"`
}

// Transaction matches the /bronze/transactions response items.
type Transaction struct {
	TransactionHash string `json:"transaction_hash"`
	SourceAccount   string `json:"source_account"`
	LedgerSequence  int64  `json:"ledger_sequence"`
	AccountSequence int64  `json:"account_sequence"`
	MaxFee          int64  `json:"max_fee"`
	OperationCount  int    `json:"operation_count"`
	Successful      bool   `json:"successful"`
	CreatedAt       string `json:"created_at"`
}

// TransactionsResponse is the envelope for /bronze/transactions.
type TransactionsResponse struct {
	Count        int           `json:"count"`
	Start        int64         `json:"start"`
	End          int64         `json:"end"`
	Transactions []Transaction `json:"transactions"`
}

// Contract matches /silver/contracts/top response items.
type Contract struct {
	ContractID   string `json:"contract_id"`
	TotalCalls   int64  `json:"total_calls"`
	UniqueCalls  int64  `json:"unique_callers"`
	TopFunction  string `json:"top_function"`
	UnknownCalls int64  `json:"unknown_calls"`
	LastActivity string `json:"last_activity"`
}

// ContractsResponse is the envelope for /silver/contracts/top.
type ContractsResponse struct {
	Count     int        `json:"count"`
	Period    string     `json:"period"`
	Contracts []Contract `json:"contracts"`
}

// Payment matches /silver/payments response items.
type Payment struct {
	TransactionHash string `json:"transaction_hash"`
	OperationID     int    `json:"operation_id"`
	LedgerSequence  int64  `json:"ledger_sequence"`
	LedgerClosedAt  string `json:"ledger_closed_at"`
	SourceAccount   string `json:"source_account"`
	Type            int    `json:"type"`
	TypeName        string `json:"type_name"`
	Destination     string `json:"destination"`
	Amount          string `json:"amount"`
	TxSuccessful    bool   `json:"tx_successful"`
	TxFeeCharged    int64  `json:"tx_fee_charged"`
	IsPaymentOp     bool   `json:"is_payment_op"`
	IsSorobanOp     bool   `json:"is_soroban_op"`
}

// PaymentsResponse is the envelope for /silver/payments.
type PaymentsResponse struct {
	Count    int       `json:"count"`
	Cursor   string    `json:"cursor,omitempty"`
	HasMore  bool      `json:"has_more"`
	Payments []Payment `json:"payments"`
	Meta     *Meta     `json:"_meta,omitempty"`
}

// Meta holds response metadata.
type Meta struct {
	ScannedLedger    int64            `json:"scanned_ledger"`
	AvailableLedgers AvailableLedgers `json:"available_ledgers"`
}

type AvailableLedgers struct {
	Oldest int64 `json:"oldest"`
	Latest int64 `json:"latest"`
}

// --- Phase 2: Transaction Detail ---

// TxFull matches /silver/tx/{hash}/full response.
type TxFull struct {
	Transaction      TxInfo             `json:"transaction"`
	Summary          TxSummary          `json:"summary"`
	Operations       []DecodedOperation `json:"operations"`
	Events           []UnifiedEvent     `json:"events"`
	SorobanResources *SorobanResources  `json:"soroban_resources"`
	CallGraph        any                `json:"call_graph"`
	Contracts        any                `json:"contracts_involved"`
}

type TxInfo struct {
	TxHash          string `json:"tx_hash"`
	Fee             int64  `json:"fee"`
	MaxFee          int64  `json:"max_fee"`
	LedgerSequence  int64  `json:"ledger_sequence"`
	OperationCount  int    `json:"operation_count"`
	Successful      bool   `json:"successful"`
	ClosedAt        string `json:"closed_at"`
	SourceAccount   string `json:"source_account"`
	AccountSequence int64  `json:"account_sequence"`
}

type SorobanResources struct {
	Instructions int64 `json:"instructions"`
	ReadBytes    int64 `json:"read_bytes"`
	WriteBytes   int64 `json:"write_bytes"`
}

type TxSummary struct {
	Description       string          `json:"description"`
	Type              string          `json:"type"` // transfer, mint, burn, swap, contract_call, classic
	InvolvedContracts []string        `json:"involved_contracts"`
	Transfer          *TransferDetail `json:"transfer,omitempty"`
	Swap              *SwapDetail     `json:"swap,omitempty"`
	Mint              *TransferDetail `json:"mint,omitempty"`
	Burn              *TransferDetail `json:"burn,omitempty"`
}

type TransferDetail struct {
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
}

type SwapDetail struct {
	AmountIn  string `json:"amount_in"`
	AmountOut string `json:"amount_out"`
	AssetIn   string `json:"asset_in"`
	AssetOut  string `json:"asset_out"`
}

type DecodedOperation struct {
	Index         int    `json:"index"`
	Type          int    `json:"type"`
	TypeName      string `json:"type_name"`
	SourceAccount string `json:"source_account"`
	ContractID    string `json:"contract_id,omitempty"`
	FunctionName  string `json:"function_name,omitempty"`
	IsSorobanOp   bool   `json:"is_soroban_op"`
	Destination   string `json:"destination,omitempty"`
	Amount        string `json:"amount,omitempty"`
	AssetCode     string `json:"asset_code,omitempty"`
	ArgumentsJSON string `json:"arguments_json,omitempty"`
}

type UnifiedEvent struct {
	EventID        string `json:"event_id"`
	LedgerSequence int64  `json:"ledger_sequence"`
	TxHash         string `json:"tx_hash"`
	ClosedAt       string `json:"closed_at"`
	EventType      string `json:"event_type"`  // transfer, mint, burn
	SourceType     string `json:"source_type"` // classic, soroban
	OperationType  int    `json:"operation_type"`
	EventIndex     int    `json:"event_index"`
	From           string `json:"from,omitempty"`
	To             string `json:"to,omitempty"`
	Amount         string `json:"amount,omitempty"`
	AssetCode      string `json:"asset_code,omitempty"`
	AssetIssuer    string `json:"asset_issuer,omitempty"`
	ContractID     string `json:"contract_id,omitempty"`
	// Token metadata inlined by the silver enricher (tx/full and tx/decoded paths only).
	// May be empty for custom Soroban contracts the API can't classify.
	TokenName     string `json:"token_name,omitempty"`
	TokenSymbol   string `json:"token_symbol,omitempty"`
	TokenDecimals int    `json:"token_decimals,omitempty"`
	TokenType     string `json:"token_type,omitempty"` // "sac", "custom_soroban"
}

// TxDiffs matches /silver/tx/{hash}/diffs response.
type TxDiffs struct {
	TxHash         string            `json:"tx_hash"`
	LedgerSequence int64             `json:"ledger_sequence"`
	BalanceChanges []TxBalanceChange `json:"balance_changes"`
	StateChanges   []TxStateChange   `json:"state_changes"`
}

type TxBalanceChange struct {
	Address     string `json:"address"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Delta       string `json:"delta"`
}

type TxStateChange struct {
	Type      string `json:"type"`
	EntryType string `json:"entry_type"`
	Key       string `json:"key"`
	Before    string `json:"before"`
	After     string `json:"after"`
}

// TxReceipt matches /silver/tx/{hash}/receipt response.
type TxReceipt struct {
	TxHash           string                      `json:"tx_hash"`
	LedgerSequence   int64                       `json:"ledger_sequence"`
	CreatedAt        string                      `json:"created_at"`
	SourceAccount    string                      `json:"source_account"`
	Successful       bool                        `json:"successful"`
	OperationCount   int                         `json:"operation_count"`
	TxType           string                      `json:"tx_type"`
	InvolvedAccounts []string                    `json:"involved_accounts"`
	Full             TxReceiptFull               `json:"full"`
	Semantic         SemanticTransactionResponse `json:"semantic"`
	Effects          []TxReceiptEffect           `json:"effects"`
	Diffs            []TxReceiptDiff             `json:"diffs"`
	Events           []UnifiedEvent              `json:"events"`
	MaterializedAt   string                      `json:"materialized_at"`
	SourceVersion    string                      `json:"source_version"`
}

type TxReceiptFull struct {
	TxHash                       string               `json:"tx_hash"`
	CreatedAt                    string               `json:"created_at"`
	Operations                   []TxReceiptOperation `json:"operations"`
	Successful                   bool                 `json:"successful"`
	SourceAccount                string               `json:"source_account"`
	LedgerSequence               int64                `json:"ledger_sequence"`
	Fee                          int64                `json:"fee"`
	MaxFee                       int64                `json:"max_fee"`
	AccountSequence              int64                `json:"account_sequence"`
	Summary                      TxSummary            `json:"summary"`
	Events                       []UnifiedEvent       `json:"events"`
	SorobanResourcesReadBytes    *int64               `json:"soroban_resources_read_bytes"`
	SorobanResourcesWriteBytes   *int64               `json:"soroban_resources_write_bytes"`
	SorobanResourcesInstructions *int64               `json:"soroban_resources_instructions"`
}

type TxReceiptOperation struct {
	Type           int    `json:"type"`
	Index          int    `json:"index,omitempty"`
	TypeName       string `json:"type_name"`
	AssetCode      string `json:"asset_code,omitempty"`
	SourceAccount  string `json:"source_account,omitempty"`
	ContractID     string `json:"contract_id,omitempty"`
	FunctionName   string `json:"function_name,omitempty"`
	IsSorobanOp    bool   `json:"is_soroban_op,omitempty"`
	Destination    string `json:"destination,omitempty"`
	Amount         string `json:"amount,omitempty"`
	OperationIndex int    `json:"operation_index,omitempty"`
}

type TxReceiptEffect struct {
	LedgerSequence   int64          `json:"ledger_sequence,omitempty"`
	TransactionHash  string         `json:"transaction_hash,omitempty"`
	OperationIndex   int            `json:"operation_index"`
	EffectIndex      int            `json:"effect_index"`
	OperationID      int64          `json:"operation_id,omitempty"`
	EffectType       any            `json:"effect_type,omitempty"`
	EffectTypeString string         `json:"effect_type_string,omitempty"`
	AccountID        string         `json:"account_id"`
	Asset            *EffectAsset   `json:"asset,omitempty"`
	Amount           string         `json:"amount,omitempty"`
	Details          map[string]any `json:"details,omitempty"`
	Timestamp        string         `json:"timestamp,omitempty"`
}

type TxReceiptDiff struct {
	Asset   string `json:"asset"`
	Delta   string `json:"delta"`
	Account string `json:"account"`
}

// --- Phase 2: Operations ---

// Operation matches /bronze/operations response items.
type Operation struct {
	TransactionHash string `json:"transaction_hash"`
	OperationID     int    `json:"operation_id"`
	LedgerSequence  int64  `json:"ledger_sequence"`
	SourceAccount   string `json:"source_account"`
	Type            int    `json:"type"`
	TypeName        string `json:"type_name"`
	LedgerClosedAt  string `json:"ledger_closed_at"`
	TxSuccessful    bool   `json:"tx_successful"`
	TxFeeCharged    int64  `json:"tx_fee_charged"`
	IsPaymentOp     bool   `json:"is_payment_op"`
	IsSorobanOp     bool   `json:"is_soroban_op"`
	SorobanContract string `json:"soroban_contract_id,omitempty"`
	SorobanFunction string `json:"soroban_function,omitempty"`
	Destination     string `json:"destination,omitempty"`
	Amount          string `json:"amount,omitempty"`
}

type OperationsResponse struct {
	Count      int         `json:"count"`
	Operations []Operation `json:"operations"`
}

// --- Phase 3: Account ---

// AccountOverview matches /silver/explorer/account response.
type AccountOverview struct {
	Account          AccountInfo     `json:"account"`
	OperationsCount  int             `json:"operations_count"`
	TransfersCount   int             `json:"transfers_count"`
	RecentOperations []EnrichedOp    `json:"recent_operations"`
	RecentTransfers  []TransferEvent `json:"recent_transfers"`
}

type AccountInfo struct {
	AccountID          string `json:"account_id"`
	Balance            string `json:"balance"`
	SequenceNumber     string `json:"sequence_number"`
	NumSubentries      int    `json:"num_subentries"`
	LastModifiedLedger int64  `json:"last_modified_ledger"`
	UpdatedAt          string `json:"updated_at"`
	CreatedAt          string `json:"created_at"`
}

type EnrichedOp struct {
	TransactionHash string `json:"transaction_hash"`
	OperationID     int    `json:"operation_id"`
	LedgerSequence  int64  `json:"ledger_sequence"`
	LedgerClosedAt  string `json:"ledger_closed_at"`
	SourceAccount   string `json:"source_account"`
	Type            int    `json:"type"`
	TypeName        string `json:"type_name"`
	TxSuccessful    bool   `json:"tx_successful"`
	TxFeeCharged    int64  `json:"tx_fee_charged"`
	IsPaymentOp     bool   `json:"is_payment_op"`
	IsSorobanOp     bool   `json:"is_soroban_op"`
	SorobanContract string `json:"soroban_contract_id,omitempty"`
	SorobanFunction string `json:"soroban_function,omitempty"`
	Destination     string `json:"destination,omitempty"`
	Amount          string `json:"amount,omitempty"`
}

// AccountBalances matches /silver/accounts/{id}/balances response.
type AccountBalances struct {
	AccountID     string         `json:"account_id"`
	Balances      []AccountAsset `json:"balances"`
	TotalBalances int            `json:"total_balances"`
}

type AccountAsset struct {
	AssetCode      string `json:"asset_code"`
	AssetIssuer    string `json:"asset_issuer"`
	AssetType      string `json:"asset_type"`
	Balance        string `json:"balance"`
	BalanceStroops int64  `json:"balance_stroops"`
	IsAuthorized   bool   `json:"is_authorized"`
	Limit          string `json:"limit"`
}

// AccountSigners matches /silver/accounts/signers response.
type AccountSignersResp struct {
	AccountID  string           `json:"account_id"`
	Signers    []AccountSigner  `json:"signers"`
	Thresholds SignerThresholds `json:"thresholds"`
}

type AccountSigner struct {
	Key     string `json:"key"`
	Type    string `json:"type"`
	Weight  int    `json:"weight"`
	Sponsor string `json:"sponsor,omitempty"`
}

type SignerThresholds struct {
	Low    int `json:"low_threshold"`
	Medium int `json:"med_threshold"`
	High   int `json:"high_threshold"`
}

// --- Phase 4: Contract Analytics ---

// ContractAnalytics matches /silver/contracts/{id}/analytics response.
type ContractAnalytics struct {
	ContractID   string             `json:"contract_id"`
	Stats        ContractCallStats  `json:"stats"`
	Timeline     ContractTimeline   `json:"timeline"`
	TopFunctions []ContractFunction `json:"top_functions"`
	DailyCalls7D []DailyCallCount   `json:"daily_calls_7d"`
}

type ContractCallStats struct {
	TotalCallsAsCaller int64 `json:"total_calls_as_caller"`
	TotalCallsAsCallee int64 `json:"total_calls_as_callee"`
	UniqueCallers      int64 `json:"unique_callers"`
	UniqueCallees      int64 `json:"unique_callees"`
}

type ContractTimeline struct {
	FirstSeen    string `json:"first_seen"`
	LastActivity string `json:"last_activity"`
}

type ContractFunction struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type DailyCallCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// ContractMetadata matches /silver/contracts/{id}/metadata response.
type ContractMetadata struct {
	ContractID                  string                             `json:"contract_id"`
	DisplayName                 string                             `json:"display_name"`
	ContractType                string                             `json:"contract_type"`
	CreatorAddress              string                             `json:"creator_address"`
	WASMHash                    string                             `json:"wasm_hash"`
	CreatedLedger               int64                              `json:"created_ledger"`
	CreatedAt                   string                             `json:"created_at"`
	TotalEntries                int64                              `json:"total_entries"`
	PersistentEntries           int64                              `json:"persistent_entries"`
	TotalStateSizeBytes         int64                              `json:"total_state_size_bytes"`
	EstimatedMonthlyRentStroops int64                              `json:"estimated_monthly_rent_stroops"`
	ExportedFunctions           []ContractExportedFunctionMetadata `json:"exported_functions"`
}

type ContractExportedFunctionMetadata struct {
	Name      string `json:"name"`
	CallCount int64  `json:"call_count"`
}

// ContractStorageResponse matches /silver/contracts/{id}/storage response.
type ContractStorageResponse struct {
	Entries []ContractStorageEntry `json:"entries"`
}

type ContractStorageEntry struct {
	Key                string          `json:"key"`
	KeyHash            string          `json:"key_hash"`
	Type               string          `json:"type"`
	Durability         string          `json:"durability"`
	SizeBytes          int64           `json:"size_bytes"`
	LiveUntilLedgerSeq int64           `json:"live_until_ledger_seq"`
	TTLRemaining       int64           `json:"ttl_remaining"`
	Expired            bool            `json:"expired"`
	DataValue          string          `json:"data_value"`
	KeyDecoded         json.RawMessage `json:"key_decoded"`
	ValueDecoded       json.RawMessage `json:"value_decoded"`
	LastModifiedLedger int64           `json:"last_modified_ledger"`
}

// ContractRecentCalls matches /silver/contracts/{id}/recent-calls response.
type ContractRecentCall struct {
	TransactionHash string `json:"transaction_hash"`
	LedgerSequence  int64  `json:"ledger_sequence"`
	ClosedAt        string `json:"closed_at"`
	SourceAccount   string `json:"source_account"`
	FunctionName    string `json:"function_name,omitempty"`
	Successful      bool   `json:"successful"`
}

// --- Phase 4: Assets ---

// AssetSummary matches /silver/assets response items.
type AssetSummary struct {
	AssetCode          string  `json:"asset_code"`
	AssetIssuer        string  `json:"asset_issuer"`
	AssetType          string  `json:"asset_type"`
	HolderCount        int     `json:"holder_count"`
	CirculatingSupply  string  `json:"circulating_supply"`
	Volume24H          string  `json:"volume_24h"`
	Transfers24H       int     `json:"transfers_24h"`
	Top10Concentration float64 `json:"top_10_concentration"`
	FirstSeen          string  `json:"first_seen"`
	LastActivity       string  `json:"last_activity"`
}

type AssetsResponse struct {
	Assets      []AssetSummary `json:"assets"`
	TotalAssets int            `json:"total_assets"`
	Cursor      string         `json:"cursor,omitempty"`
	HasMore     bool           `json:"has_more"`
	GeneratedAt string         `json:"generated_at"`
}

// AssetDetail matches /silver/assets/{asset} — the canonical detail endpoint for
// native assets, classic assets, and token-contract asset identifiers.
type AssetDetail struct {
	CanonicalSlug       string               `json:"canonical_slug,omitempty"`
	Route               string               `json:"route,omitempty"`
	AssetCode           string               `json:"asset_code,omitempty"`
	AssetIssuer         string               `json:"asset_issuer,omitempty"`
	AssetType           string               `json:"asset_type,omitempty"`
	DisplayName         string               `json:"display_name,omitempty"`
	Name                string               `json:"name,omitempty"`
	Symbol              string               `json:"symbol,omitempty"`
	Decimals            int                  `json:"decimals,omitempty"`
	TokenType           string               `json:"token_type,omitempty"`
	ContractID          string               `json:"contract_id,omitempty"`
	HomeDomain          string               `json:"home_domain,omitempty"`
	TomlVerified        bool                 `json:"toml_verified,omitempty"`
	AuthRequired        bool                 `json:"auth_required,omitempty"`
	AuthRevocable       bool                 `json:"auth_revocable,omitempty"`
	AuthImmutable       bool                 `json:"auth_immutable,omitempty"`
	AuthClawback        bool                 `json:"auth_clawback_enabled,omitempty"`
	HolderCount         int64                `json:"holder_count,omitempty"`
	TrustlineCount      int64                `json:"trustline_count,omitempty"`
	TotalTrustlines     int64                `json:"total_trustlines,omitempty"`
	CirculatingSupply   string               `json:"circulating_supply,omitempty"`
	Volume24H           string               `json:"volume_24h,omitempty"`
	Transfers24H        int64                `json:"transfers_24h,omitempty"`
	UniqueAccounts24H   int64                `json:"unique_accounts_24h,omitempty"`
	Top10Concentration  float64              `json:"top_10_concentration,omitempty"`
	Top100Concentration float64              `json:"top_100_concentration,omitempty"`
	LinkedTokenContract string               `json:"linked_token_contract,omitempty"`
	LinkedContractID    string               `json:"linked_contract_id,omitempty"`
	LinkedTokenType     string               `json:"linked_token_type,omitempty"`
	LinkedClassicAsset  string               `json:"linked_classic_asset,omitempty"`
	Asset               *AssetIdentity       `json:"asset,omitempty"`
	Issuer              *AssetIssuerInfo     `json:"issuer,omitempty"`
	Stats               *AssetStatsEnvelope  `json:"stats,omitempty"`
	LinkedTokens        []AssetLinkSummary   `json:"linked_tokens,omitempty"`
	RecentTransfers     []AssetTransferBrief `json:"recent_transfers,omitempty"`
	TransferPreview     []AssetTransferBrief `json:"transfer_preview,omitempty"`
	TopPairs            []AssetPairSummary   `json:"top_pairs,omitempty"`
	PairsPreview        []AssetPairSummary   `json:"pairs_preview,omitempty"`
	TopHolders          []AssetHolderSummary `json:"top_holders,omitempty"`
	HoldersPreview      []AssetHolderSummary `json:"holders_preview,omitempty"`
	Links               []AssetLinkSummary   `json:"links,omitempty"`
	LinksPreview        []AssetLinkSummary   `json:"links_preview,omitempty"`
}

type AssetIdentity struct {
	Code   string `json:"code,omitempty"`
	Issuer string `json:"issuer,omitempty"`
	Type   string `json:"type,omitempty"`
}

type AssetIssuerInfo struct {
	AccountID     string `json:"account_id,omitempty"`
	HomeDomain    string `json:"home_domain,omitempty"`
	AuthRequired  bool   `json:"auth_required,omitempty"`
	AuthRevocable bool   `json:"auth_revocable,omitempty"`
	AuthImmutable bool   `json:"auth_immutable,omitempty"`
	AuthClawback  bool   `json:"auth_clawback_enabled,omitempty"`
}

type AssetTransferBrief struct {
	Timestamp             string `json:"timestamp,omitempty"`
	TransactionHash       string `json:"transaction_hash,omitempty"`
	LedgerSequence        int64  `json:"ledger_sequence,omitempty"`
	SourceType            string `json:"source_type,omitempty"`
	FromAccount           string `json:"from_account,omitempty"`
	ToAccount             string `json:"to_account,omitempty"`
	AssetCode             string `json:"asset_code,omitempty"`
	TokenContractID       string `json:"token_contract_id,omitempty"`
	Amount                string `json:"amount,omitempty"`
	TransactionSuccessful bool   `json:"transaction_successful,omitempty"`
}

type AssetPairSummary struct {
	CounterAsset string `json:"counter_asset,omitempty"`
	CounterCode  string `json:"counter_code,omitempty"`
	Pool         string `json:"pool,omitempty"`
	Liquidity    string `json:"liquidity,omitempty"`
	Volume24H    string `json:"volume_24h,omitempty"`
}

type AssetHolderSummary struct {
	Account  string  `json:"account,omitempty"`
	Balance  string  `json:"balance,omitempty"`
	SharePct float64 `json:"share_pct,omitempty"`
}

type AssetLinkSummary struct {
	Relation      string `json:"relation,omitempty"`
	Label         string `json:"label,omitempty"`
	CanonicalSlug string `json:"canonical_slug,omitempty"`
	Route         string `json:"route,omitempty"`
	AssetCode     string `json:"asset_code,omitempty"`
	AssetIssuer   string `json:"asset_issuer,omitempty"`
	ContractID    string `json:"contract_id,omitempty"`
	TokenType     string `json:"token_type,omitempty"`
	TokenName     string `json:"token_name,omitempty"`
	TokenSymbol   string `json:"token_symbol,omitempty"`
	TokenDecimals int    `json:"token_decimals,omitempty"`
}

type AssetLinksResponse struct {
	Asset        string             `json:"asset,omitempty"`
	Links        []AssetLinkSummary `json:"links"`
	LinkedTokens []AssetLinkSummary `json:"linked_tokens,omitempty"`
}

type AssetPairsResponse struct {
	Asset string             `json:"asset,omitempty"`
	Pairs []AssetPairSummary `json:"pairs"`
}

type AssetHoldersResponse struct {
	Asset   string               `json:"asset,omitempty"`
	Holders []AssetHolderSummary `json:"holders"`
	Count   int                  `json:"count,omitempty"`
	HasMore bool                 `json:"has_more,omitempty"`
	Cursor  string               `json:"cursor,omitempty"`
}

type AssetStatsEnvelope struct {
	Asset       *AssetIdentity `json:"asset,omitempty"`
	Stats       *AssetStats    `json:"stats,omitempty"`
	GeneratedAt string         `json:"generated_at,omitempty"`
}

type AssetStats struct {
	Asset               *AssetIdentity `json:"asset,omitempty"`
	HolderCount         int64          `json:"holder_count,omitempty"`
	TotalHolders        int64          `json:"total_holders,omitempty"`
	TrustlineCount      int64          `json:"trustline_count,omitempty"`
	TotalTrustlines     int64          `json:"total_trustlines,omitempty"`
	CirculatingSupply   string         `json:"circulating_supply,omitempty"`
	Volume24H           string         `json:"volume_24h,omitempty"`
	Transfers24H        int64          `json:"transfers_24h,omitempty"`
	UniqueAccounts24H   int64          `json:"unique_accounts_24h,omitempty"`
	Top10Concentration  float64        `json:"top_10_concentration,omitempty"`
	Top100Concentration float64        `json:"top_100_concentration,omitempty"`
}

// TransferEvent matches /silver/transfers response items.
type TransferEvent struct {
	Timestamp             string `json:"timestamp"`
	TransactionHash       string `json:"transaction_hash"`
	LedgerSequence        int64  `json:"ledger_sequence"`
	SourceType            string `json:"source_type"` // classic, soroban
	FromAccount           string `json:"from_account"`
	ToAccount             string `json:"to_account"`
	AssetCode             string `json:"asset_code"`
	TokenContractID       string `json:"token_contract_id,omitempty"`
	Amount                string `json:"amount"`
	TransactionSuccessful bool   `json:"transaction_successful"`
}

type TransfersResponse struct {
	Count     int             `json:"count"`
	Cursor    string          `json:"cursor,omitempty"`
	HasMore   bool            `json:"has_more"`
	Transfers []TransferEvent `json:"transfers"`
	Meta      *Meta           `json:"_meta,omitempty"`
}

// --- Phase 5: Events + Search ---

// EventsResponse matches /silver/events response.
type EventsResponse struct {
	Count      int            `json:"count"`
	Events     []UnifiedEvent `json:"events"`
	HasMore    bool           `json:"has_more"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// SearchResult matches /silver/search response.
type SearchResults struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	Type    string         `json:"type"` // account, contract, transaction, ledger, asset
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Details map[string]any `json:"details,omitempty"`
}

// --- Smart Wallet Detection ---

// SmartWalletInfo matches /silver/smart-wallet/{contract_id} response.
type SmartWalletInfo struct {
	ContractID     string              `json:"contract_id"`
	IsSmartWallet  bool                `json:"is_smart_wallet"`
	WalletType     string              `json:"wallet_type,omitempty"`    // "crossmint", "openzeppelin", "sep50", ""
	Implementation string              `json:"implementation,omitempty"` // more specific impl name
	HasCheckAuth   bool                `json:"has_check_auth"`
	Confidence     float64             `json:"confidence,omitempty"`
	SignerCount    int                 `json:"signer_count"`
	Signers        []SmartWalletSigner `json:"signers,omitempty"`
}

type SmartWalletSigner struct {
	ID      string `json:"id"`
	KeyType string `json:"key_type"` // "ed25519", "secp256r1", "webauthn"
}

// SmartWalletDetail matches /silver/smart-wallets/{contract_id} response.
type SmartWalletDetail struct {
	ContractID    string                           `json:"contract_id"`
	DisplayName   string                           `json:"display_name,omitempty"`
	IsSmartWallet bool                             `json:"is_smart_wallet"`
	Meta          SmartWalletDetailMeta            `json:"meta"`
	Wallet        SmartWalletDetailWallet          `json:"wallet"`
	Account       SmartWalletDetailAccount         `json:"account"`
	SignerConfig  SmartWalletDetailSignerConfig    `json:"signer_config"`
	Policies      SmartWalletDetailPolicies        `json:"policies"`
	SessionKeys   SmartWalletDetailSessionKeys     `json:"session_keys"`
	Contract      SmartWalletDetailContract        `json:"contract"`
	Rent          SmartWalletDetailRent            `json:"rent"`
	Activity      SmartWalletDetailActivitySummary `json:"activity_summary"`
	Timeline      []SmartWalletDetailTimelineItem  `json:"timeline"`
}

type SmartWalletDetailMeta struct {
	GeneratedAt string   `json:"generated_at,omitempty"`
	Sources     []string `json:"sources,omitempty"`
	Partial     bool     `json:"partial"`
}

type SmartWalletDetailWallet struct {
	WalletType           string   `json:"wallet_type,omitempty"`
	Implementation       string   `json:"implementation,omitempty"`
	ClassificationSource string   `json:"classification_source,omitempty"`
	Confidence           float64  `json:"confidence,omitempty"`
	HasCheckAuth         bool     `json:"has_check_auth"`
	Evidence             []string `json:"evidence,omitempty"`
}

type SmartWalletDetailAccount struct {
	CreatedAt     string                     `json:"created_at,omitempty"`
	CreatedLedger int64                      `json:"created_ledger,omitempty"`
	LastActivity  string                     `json:"last_activity_at,omitempty"`
	Balances      []SmartWalletDetailBalance `json:"balances,omitempty"`
	Source        string                     `json:"source,omitempty"`
}

type SmartWalletDetailBalance struct {
	AssetCode string `json:"asset_code,omitempty"`
	AssetType string `json:"asset_type,omitempty"`
	Balance   string `json:"balance,omitempty"`
	ValueUSD  any    `json:"value_usd,omitempty"`
}

type SmartWalletDetailSignerConfig struct {
	Decoded            bool                           `json:"decoded"`
	Source             string                         `json:"source,omitempty"`
	SignerCount        int                            `json:"signer_count"`
	Signers            []SmartWalletDetailSigner      `json:"signers,omitempty"`
	ApprovalModel      SmartWalletDetailApprovalModel `json:"approval_model"`
	Thresholds         *SmartWalletDetailThresholds   `json:"thresholds,omitempty"`
	RequiredWeight     *int                           `json:"required_weight,omitempty"`
	TotalWeight        *int                           `json:"total_weight,omitempty"`
	MinSignersEstimate *int                           `json:"min_signers_estimate,omitempty"`
}

type SmartWalletDetailSigner struct {
	ID      string `json:"id,omitempty"`
	KeyType string `json:"key_type,omitempty"`
	Role    string `json:"role,omitempty"`
	Label   string `json:"label,omitempty"`
	Weight  *int   `json:"weight,omitempty"`
}

type SmartWalletDetailThresholds struct {
	Low    *int `json:"low,omitempty"`
	Medium *int `json:"medium,omitempty"`
	High   *int `json:"high,omitempty"`
}

type SmartWalletDetailApprovalModel struct {
	Type    string `json:"type,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type SmartWalletDetailPolicies struct {
	Decoded bool                      `json:"decoded"`
	Items   []SmartWalletDetailPolicy `json:"items,omitempty"`
}

type SmartWalletDetailPolicy struct {
	PolicyType  string                             `json:"policy_type,omitempty"`
	Name        string                             `json:"name,omitempty"`
	Description string                             `json:"description,omitempty"`
	Status      string                             `json:"status,omitempty"`
	Config      map[string]any                     `json:"config,omitempty"`
	Contracts   []SmartWalletDetailAllowedContract `json:"contracts,omitempty"`
}

type SmartWalletDetailAllowedContract struct {
	ContractID  string   `json:"contract_id,omitempty"`
	DisplayName string   `json:"display_name,omitempty"`
	Methods     []string `json:"methods,omitempty"`
}

type SmartWalletDetailSessionKeys struct {
	Supported bool                          `json:"supported"`
	Count     int                           `json:"count"`
	Items     []SmartWalletDetailSessionKey `json:"items,omitempty"`
}

type SmartWalletDetailSessionKey struct {
	Key        string `json:"key,omitempty"`
	KeyType    string `json:"key_type,omitempty"`
	Scope      string `json:"scope,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	SpendLimit string `json:"spend_limit,omitempty"`
	UsedAmount string `json:"used_amount,omitempty"`
	Status     string `json:"status,omitempty"`
}

type SmartWalletDetailContract struct {
	WASMHash          string   `json:"wasm_hash,omitempty"`
	Deployer          string   `json:"deployer,omitempty"`
	InterfaceType     string   `json:"interface_type,omitempty"`
	ExportedFunctions []string `json:"exported_functions,omitempty"`
	ObservedFunctions []string `json:"observed_functions,omitempty"`
	StorageEntries    int64    `json:"storage_entries,omitempty"`
	PersistentEntries int64    `json:"persistent_entries,omitempty"`
	TemporaryEntries  int64    `json:"temporary_entries,omitempty"`
	StateSizeBytes    int64    `json:"state_size_bytes,omitempty"`
}

type SmartWalletDetailRent struct {
	RentStatus                  string `json:"rent_status,omitempty"`
	TTLLedgers                  int64  `json:"ttl_ledgers,omitempty"`
	TTLExpiresAt                string `json:"ttl_expires_at,omitempty"`
	EstimatedMonthlyRentStroops int64  `json:"estimated_monthly_rent_stroops,omitempty"`
}

type SmartWalletDetailActivitySummary struct {
	TotalTransactions7D      int                              `json:"total_transactions_7d"`
	SuccessfulTransactions7D int                              `json:"successful_transactions_7d"`
	PolicyUpdates30D         int                              `json:"policy_updates_30d"`
	Approvals30D             int                              `json:"approvals_30d"`
	ProtocolInteractions30D  int                              `json:"protocol_interactions_30d"`
	UniqueCallers30D         int                              `json:"unique_callers_30d"`
	ActiveWindows            []string                         `json:"active_windows,omitempty"`
	CommonFunctions          []SmartWalletDetailFunctionCount `json:"common_functions,omitempty"`
	CommonProtocols          []SmartWalletDetailProtocol      `json:"common_protocols,omitempty"`
}

type SmartWalletDetailFunctionCount struct {
	Name  string `json:"name,omitempty"`
	Count int64  `json:"count"`
}

type SmartWalletDetailProtocol struct {
	ContractID       string `json:"contract_id,omitempty"`
	DisplayName      string `json:"display_name,omitempty"`
	InteractionCount int64  `json:"interaction_count"`
}

type SmartWalletDetailTimelineItem struct {
	Type           string   `json:"type,omitempty"`
	Subtype        string   `json:"subtype,omitempty"`
	Timestamp      string   `json:"timestamp,omitempty"`
	LedgerSequence int64    `json:"ledger_sequence,omitempty"`
	TxHash         string   `json:"tx_hash,omitempty"`
	Successful     bool     `json:"successful"`
	Title          string   `json:"title,omitempty"`
	Description    string   `json:"description,omitempty"`
	Actors         []string `json:"actors,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
}

// --- Fee Stats ---

// FeeStats matches /silver/stats/fees response.
type FeeStats struct {
	Period      string  `json:"period"`
	MedianFee   int64   `json:"median_fee"`
	P75Fee      int64   `json:"p75_fee"`
	P90Fee      int64   `json:"p90_fee"`
	P99Fee      int64   `json:"p99_fee"`
	MinFee      int64   `json:"min_fee"`
	MaxFee      int64   `json:"max_fee"`
	TotalFees   int64   `json:"total_fees"`
	TxCount     int64   `json:"tx_count"`
	SurgeActive bool    `json:"surge_active"`
	SurgePct    float64 `json:"surge_pct_of_ledgers,omitempty"`
	GeneratedAt string  `json:"generated_at"`
}

// --- Soroban Stats ---

// SorobanStats matches /silver/stats/soroban response.
type SorobanStats struct {
	Contracts   SorobanContractStats  `json:"contracts"`
	Execution   SorobanExecutionStats `json:"execution"`
	State       SorobanStateStats     `json:"state"`
	GeneratedAt string                `json:"generated_at"`
}

type SorobanContractStats struct {
	TotalDeployed int64 `json:"total_deployed"`
	Active24H     int64 `json:"active_24h"`
	Active7D      int64 `json:"active_7d"`
}

type SorobanExecutionStats struct {
	TotalInvocations24H  int64 `json:"total_invocations_24h"`
	AvgCPUInsns          int64 `json:"avg_cpu_insns,omitempty"`
	TotalCPUInsns        int64 `json:"total_cpu_insns,omitempty"`
	RentBurned24HStroops int64 `json:"rent_burned_24h_stroops,omitempty"`
}

type SorobanStateStats struct {
	PersistentEntries int64 `json:"persistent_entries"`
	TemporaryEntries  int64 `json:"temporary_entries"`
}

// --- Semantic Contracts ---

// SemanticContract matches /semantic/contracts response items.
type SemanticContract struct {
	ContractID        string   `json:"contract_id"`
	ContractType      string   `json:"contract_type"`
	WalletType        *string  `json:"wallet_type,omitempty"` // "crossmint", "openzeppelin", "sep50", nil
	TokenName         *string  `json:"token_name,omitempty"`
	TokenSymbol       *string  `json:"token_symbol,omitempty"`
	TokenDecimals     *int     `json:"token_decimals,omitempty"`
	DeployerAccount   *string  `json:"deployer_account,omitempty"`
	DeployedAt        *string  `json:"deployed_at,omitempty"`
	DeployedLedger    *int64   `json:"deployed_ledger,omitempty"`
	TotalInvocations  int64    `json:"total_invocations"`
	LastActivity      *string  `json:"last_activity,omitempty"`
	UniqueCallers     int64    `json:"unique_callers"`
	ObservedFunctions []string `json:"observed_functions,omitempty"`
}

// --- Semantic Transactions ---

// SemanticTransactionResponse matches /silver/tx/{hash}/semantic response.
type SemanticTransactionResponse struct {
	Transaction    SemanticTransactionInfo           `json:"transaction"`
	Classification SemanticTransactionClassification `json:"classification"`
	Actors         []SemanticActor                   `json:"actors"`
	Assets         SemanticAssetContext              `json:"assets"`
	Operations     []DecodedOperation                `json:"operations"`
	Events         []UnifiedEvent                    `json:"events"`
	Diffs          *TxDiffs                          `json:"diffs,omitempty"`
	CallGraph      []SemanticCallEdge                `json:"call_graph,omitempty"`
	LegacySummary  TxSummary                         `json:"legacy_summary"`
}

type SemanticTransactionInfo struct {
	TxHash          string  `json:"tx_hash"`
	LedgerSequence  int64   `json:"ledger_sequence"`
	ClosedAt        string  `json:"closed_at"`
	Successful      bool    `json:"successful"`
	Fee             int64   `json:"fee"`
	OperationCount  int     `json:"operation_count"`
	SourceAccount   *string `json:"source_account,omitempty"`
	AccountSequence *int64  `json:"account_sequence,omitempty"`
	MaxFee          *int64  `json:"max_fee,omitempty"`
}

type SemanticTransactionClassification struct {
	TxType             string   `json:"tx_type"`
	Subtype            string   `json:"subtype,omitempty"`
	Confidence         string   `json:"confidence"`
	OperationTypes     []string `json:"operation_types,omitempty"`
	WalletInvolved     bool     `json:"wallet_involved"`
	EffectiveActorType string   `json:"effective_actor_type,omitempty"`
}

type SemanticActor struct {
	ActorID    string                 `json:"actor_id"`
	ActorType  string                 `json:"actor_type"`
	Label      *string                `json:"label,omitempty"`
	Roles      []string               `json:"roles"`
	Wallet     *SemanticWalletContext `json:"wallet,omitempty"`
	ContractID *string                `json:"contract_id,omitempty"`
}

type SemanticWalletContext struct {
	WalletType     string              `json:"wallet_type,omitempty"`
	Implementation string              `json:"implementation,omitempty"`
	Confidence     float64             `json:"confidence,omitempty"`
	HasCheckAuth   bool                `json:"has_check_auth,omitempty"`
	SignerCount    int                 `json:"signer_count,omitempty"`
	Signers        []SmartWalletSigner `json:"signers,omitempty"`
}

type SemanticAssetContext struct {
	Sent      *SemanticAssetMovement  `json:"sent,omitempty"`
	Received  *SemanticAssetMovement  `json:"received,omitempty"`
	Movements []SemanticAssetMovement `json:"movements,omitempty"`
	Path      []string                `json:"path,omitempty"`
}

type SemanticAssetMovement struct {
	Amount   string  `json:"amount"`
	Asset    string  `json:"asset"`
	From     *string `json:"from,omitempty"`
	To       *string `json:"to,omitempty"`
	Contract *string `json:"contract,omitempty"`
	Kind     string  `json:"kind,omitempty"`
}

type SemanticCallEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Function   string `json:"function"`
	Depth      int    `json:"depth"`
	Order      int    `json:"order"`
	Successful bool   `json:"successful"`
}

type SemanticContractsResponse struct {
	Contracts []SemanticContract `json:"contracts"`
	Count     int                `json:"count"`
}

// --- Contract Interface ---

// ContractInterface matches /silver/contracts/{id}/interface response.
type ContractInterface struct {
	ContractID        string   `json:"contract_id"`
	DetectedType      string   `json:"detected_type"`
	Interface         any      `json:"interface,omitempty"`
	ObservedFunctions []string `json:"observed_functions"`
}

// --- Account Activity ---

// AccountActivityItem matches /silver/accounts/{id}/activity response items.
type AccountActivityItem struct {
	Type           string         `json:"type"`
	Timestamp      string         `json:"timestamp"`
	LedgerSequence int64          `json:"ledger_sequence"`
	TxHash         string         `json:"tx_hash"`
	Details        map[string]any `json:"details,omitempty"`
}

type AccountActivityResponse struct {
	AccountID string                `json:"account_id"`
	Activity  []AccountActivityItem `json:"activity"`
	Cursor    string                `json:"cursor,omitempty"`
	HasMore   bool                  `json:"has_more"`
	Count     int                   `json:"count"`
}

// --- Account Offers ---

// AccountOffer matches /silver/accounts/{id}/offers response items.
type AccountOffer struct {
	OfferID      int64  `json:"offer_id"`
	SellerID     string `json:"seller_id"`
	SellingAsset string `json:"selling_asset"`
	BuyingAsset  string `json:"buying_asset"`
	Amount       string `json:"amount"`
	Price        string `json:"price"`
}

type AccountOffersResponse struct {
	AccountID string         `json:"account_id"`
	Offers    []AccountOffer `json:"offers"`
	Count     int            `json:"count"`
	HasMore   bool           `json:"has_more"`
}

// --- Transaction Summaries ---

// TransactionSummaryItem matches /silver/transactions/summaries response items.
type TransactionSummaryItem struct {
	TxHash          string  `json:"tx_hash"`
	LedgerSequence  int64   `json:"ledger_sequence"`
	ClosedAt        string  `json:"closed_at"`
	SourceAccount   string  `json:"source_account"`
	FeeCharged      int64   `json:"fee_charged"`
	OpCount         int64   `json:"op_count"`
	Successful      bool    `json:"successful"`
	HasSoroban      bool    `json:"has_soroban"`
	PrimaryContract *string `json:"primary_contract,omitempty"`
	TxType          string  `json:"tx_type"`
}

type TransactionSummariesResponse struct {
	Transactions []TransactionSummaryItem `json:"transactions"`
	Count        int                      `json:"count"`
}

// --- Generic Events ---

// GenericEvent matches /silver/events/generic response items.
type GenericEvent struct {
	ContractID     string `json:"contract_id"`
	EventType      string `json:"event_type"`
	LedgerSequence int64  `json:"ledger_sequence"`
	TxHash         string `json:"tx_hash"`
	ClosedAt       string `json:"closed_at"`
	TopicsDecoded  string `json:"topics_decoded,omitempty"`
	DataDecoded    string `json:"data_decoded,omitempty"`
}

type GenericEventsResponse struct {
	Events     []GenericEvent `json:"events"`
	Count      int            `json:"count"`
	HasMore    bool           `json:"has_more"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// --- Batch Decoded Transactions ---

// RecentTransactionsResponse matches /silver/transactions/recent response.
// Single-call replacement for the bronze/stats + bronze/transactions + silver/tx/batch/decoded pattern.
type RecentTransactionsResponse struct {
	LatestSequence int64                `json:"latest_sequence"`
	Count          int                  `json:"count"`
	Transactions   []DecodedTransaction `json:"transactions"`
}

// Token matches /silver/tokens/{contract_id} — metadata for a SAC or custom Soroban token.
// Used to resolve an event's contract_id into a human-readable asset symbol and decimal scale.
type Token struct {
	ContractID    string `json:"contract_id"`
	Name          string `json:"name,omitempty"`
	Symbol        string `json:"symbol,omitempty"`
	SourceType    string `json:"source_type,omitempty"` // "soroban", "classic"
	TokenType     string `json:"token_type,omitempty"`  // "sac", "custom_soroban"
	Decimals      int    `json:"decimals"`
	HolderCount   int64  `json:"holder_count,omitempty"`
	TransferCount int64  `json:"transfer_count,omitempty"`
	FirstSeen     string `json:"first_seen,omitempty"`
	LastActivity  string `json:"last_activity,omitempty"`
}

type TokenStats struct {
	Symbol         string `json:"symbol,omitempty"`
	TokenType      string `json:"token_type,omitempty"`
	HolderCount    int64  `json:"holder_count,omitempty"`
	TotalSupply    string `json:"total_supply,omitempty"`
	TotalSupplyRaw int64  `json:"total_supply_raw,omitempty"`
	Transfers24H   int64  `json:"transfers_24h,omitempty"`
	Volume24H      string `json:"volume_24h,omitempty"`
	Volume24HRaw   int64  `json:"volume_24h_raw,omitempty"`
}

type TokenTransfer struct {
	EventID        string `json:"event_id,omitempty"`
	LedgerSequence int64  `json:"ledger_sequence,omitempty"`
	TxHash         string `json:"tx_hash,omitempty"`
	ClosedAt       string `json:"closed_at,omitempty"`
	EventType      string `json:"event_type,omitempty"`
	From           string `json:"from,omitempty"`
	To             string `json:"to,omitempty"`
	Amount         string `json:"amount,omitempty"`
	SourceType     string `json:"source_type,omitempty"`
	OperationType  int    `json:"operation_type,omitempty"`
}

type TokenTransfersResponse struct {
	Transfers []TokenTransfer `json:"transfers"`
	Count     int             `json:"count,omitempty"`
	HasMore   bool            `json:"has_more,omitempty"`
	Cursor    string          `json:"cursor,omitempty"`
}

// LedgerFullResponse matches /silver/ledger/{seq}/full — a composite endpoint returning
// ledger header, transactions, operations, fee distribution, and Soroban stats in one call.
// Replaces the 6-call fan-out used by the ledger detail page.
type LedgerFullResponse struct {
	LedgerSequence int64          `json:"ledger_sequence"`
	Ledger         Ledger         `json:"ledger"`
	Transactions   []Transaction  `json:"transactions"`
	Operations     []Operation    `json:"operations"`
	Fees           *LedgerFees    `json:"fees,omitempty"`
	Soroban        *LedgerSoroban `json:"soroban,omitempty"`
	GeneratedAt    string         `json:"generated_at"`
}

// RecentLedger matches a single entry in /silver/ledgers/recent response.
// Leaner shape than the bronze Ledger type (no total_coins, base_reserve, etc.).
type RecentLedger struct {
	LedgerSequence     int64  `json:"ledger_sequence"`
	ClosedAt           string `json:"closed_at"`
	LedgerHash         string `json:"ledger_hash"`
	PreviousLedgerHash string `json:"previous_ledger_hash"`
	ProtocolVersion    int    `json:"protocol_version"`
	BaseFeeStroops     int64  `json:"base_fee_stroops"`
	SuccessfulTxCount  int    `json:"successful_tx_count"`
	FailedTxCount      int    `json:"failed_tx_count"`
	OperationCount     int    `json:"operation_count"`
}

// RecentLedgersResponse matches /silver/ledgers/recent response.
// Single-call replacement for the bronze/stats + bronze/ledgers pattern.
type RecentLedgersResponse struct {
	LatestSequence int64          `json:"latest_sequence"`
	Count          int            `json:"count"`
	Ledgers        []RecentLedger `json:"ledgers"`
}

// BatchDecodedResponse matches /silver/tx/batch/decoded response.
type BatchDecodedResponse struct {
	Transactions []DecodedTransaction `json:"transactions"`
	Count        int                  `json:"count"`
	Errors       []BatchDecodeError   `json:"errors,omitempty"`
}

type DecodedTransaction struct {
	TxHash          string             `json:"tx_hash"`
	LedgerSequence  int64              `json:"ledger_sequence"`
	ClosedAt        string             `json:"closed_at"`
	Successful      bool               `json:"successful"`
	Fee             int64              `json:"fee"`
	OperationCount  int                `json:"operation_count"`
	SourceAccount   *string            `json:"source_account,omitempty"`
	Summary         *TxSummary         `json:"summary,omitempty"`
	Operations      []DecodedOperation `json:"operations,omitempty"`
	Events          []UnifiedEvent     `json:"events,omitempty"`
	SorobanResInsns *int64             `json:"soroban_resources_instructions,omitempty"`
	SorobanResRead  *int64             `json:"soroban_resources_read_bytes,omitempty"`
	SorobanResWrite *int64             `json:"soroban_resources_write_bytes,omitempty"`
}

type BatchDecodeError struct {
	TxHash string `json:"tx_hash"`
	Error  string `json:"error"`
}

// --- Per-Ledger Stats ---

// LedgerFees matches /silver/ledgers/{seq}/fees response.
type LedgerFees struct {
	LedgerSequence int64       `json:"ledger_sequence"`
	TxCount        int         `json:"tx_count"`
	MinFee         int64       `json:"min_fee"`
	MaxFee         int64       `json:"max_fee"`
	MedianFee      int64       `json:"median_fee"`
	P90Fee         int64       `json:"p90_fee"`
	TotalFees      int64       `json:"total_fees"`
	Histogram      []FeeBucket `json:"histogram,omitempty"`
	GeneratedAt    string      `json:"generated_at"`
}

type FeeBucket struct {
	Min   int64 `json:"min"`
	Max   int64 `json:"max"`
	Count int   `json:"count"`
}

// LedgerSoroban matches /silver/ledgers/{seq}/soroban response.
type LedgerSoroban struct {
	LedgerSequence   int64  `json:"ledger_sequence"`
	SorobanTxCount   int64  `json:"soroban_tx_count"`
	TotalCPUInsns    int64  `json:"total_cpu_insns"`
	TotalReadBytes   int64  `json:"total_read_bytes"`
	TotalWriteBytes  int64  `json:"total_write_bytes"`
	TotalRentCharged int64  `json:"total_rent_charged"`
	UniqueContracts  int64  `json:"unique_contracts"`
	GeneratedAt      string `json:"generated_at"`
}

// LedgerSummary matches the older compact /silver/ledger/{seq}/summary shape.
// Kept for the ledger-first v2 page while the live home feed uses LedgerFeedSummary below.
type LedgerSummary struct {
	LedgerSequence   int64                         `json:"ledger_sequence"`
	TxCount          int64                         `json:"tx_count,omitempty"`
	TransactionCount int64                         `json:"transaction_count,omitempty"`
	Swaps            int64                         `json:"swaps,omitempty"`
	Calls            int64                         `json:"calls,omitempty"`
	Agents           int64                         `json:"agents,omitempty"`
	InstructionPct   int                           `json:"instruction_pct,omitempty"`
	ReadWritePct     int                           `json:"read_write_pct,omitempty"`
	Classifications  *LedgerSummaryClassifications `json:"classifications,omitempty"`
	Utilization      *LedgerSummaryUtilization     `json:"utilization,omitempty"`
}

type LedgerSummaryClassifications struct {
	Swaps  int64 `json:"swaps,omitempty"`
	Calls  int64 `json:"calls,omitempty"`
	Agents int64 `json:"agents,omitempty"`
}

type LedgerSummaryUtilization struct {
	InstructionPct int `json:"instruction_pct,omitempty"`
	ReadWritePct   int `json:"read_write_pct,omitempty"`
}

// LedgerFeedSummary matches the current rich /silver/ledger/{seq}/summary response used by /v2/home.
type LedgerFeedSummary struct {
	Ledger                     LedgerFeedSummaryLedger               `json:"ledger"`
	Totals                     LedgerFeedSummaryTotals               `json:"totals"`
	ClassificationCounts       LedgerFeedSummaryClassificationCounts `json:"classification_counts"`
	SorobanUtilization         LedgerFeedSummarySorobanUtilization   `json:"soroban_utilization"`
	Sampling                   LedgerFeedSummarySampling             `json:"sampling"`
	RepresentativeTransactions []LedgerFeedRepresentativeTransaction `json:"representative_transactions,omitempty"`
	Composition                LedgerFeedSummaryComposition          `json:"composition"`
	Provenance                 LedgerFeedSummaryProvenance           `json:"provenance"`
}

type LedgerFeedSummaryLedger struct {
	Sequence          int64  `json:"sequence"`
	ClosedAt          string `json:"closed_at"`
	ClosedByNodeID    string `json:"closed_by_node_id,omitempty"`
	ClosedByValidator string `json:"closed_by_validator,omitempty"`
	ProtocolVersion   int    `json:"protocol_version"`
	Hash              string `json:"hash,omitempty"`
	PreviousHash      string `json:"previous_hash,omitempty"`
}

type LedgerFeedSummaryTotals struct {
	TransactionCount   int64 `json:"transaction_count"`
	SuccessfulTxCount  int64 `json:"successful_tx_count"`
	FailedTxCount      int64 `json:"failed_tx_count"`
	OperationCount     int64 `json:"operation_count"`
	ContractEventCount int64 `json:"contract_event_count"`
	SorobanOpCount     int64 `json:"soroban_op_count"`
	TotalFeeCharged    int64 `json:"total_fee_charged"`
}

type LedgerFeedSummaryClassificationCounts struct {
	SwapTxCount         int64 `json:"swap_tx_count"`
	ContractCallTxCount int64 `json:"contract_call_tx_count"`
	ClassicTxCount      int64 `json:"classic_tx_count"`
	PaymentTxCount      int64 `json:"payment_tx_count"`
	SorobanTxCount      int64 `json:"soroban_tx_count"`
}

type LedgerFeedSummarySorobanUtilization struct {
	InstructionsUsed   int64 `json:"instructions_used"`
	ReadBytesUsed      int64 `json:"read_bytes_used"`
	WriteBytesUsed     int64 `json:"write_bytes_used"`
	ReadWriteBytesUsed int64 `json:"read_write_bytes_used"`
	InstructionPct     int   `json:"instruction_pct,omitempty"`
	ReadWritePct       int   `json:"read_write_pct,omitempty"`
}

type LedgerFeedSummarySampling struct {
	Strategy                    string `json:"strategy,omitempty"`
	SampleCount                 int    `json:"sample_count"`
	RepresentedTransactionCount int64  `json:"represented_transaction_count"`
	TotalTransactionCount       int64  `json:"total_transaction_count"`
}

type LedgerFeedRepresentativeTransaction struct {
	TxHash         string                                 `json:"tx_hash"`
	Category       string                                 `json:"category"`
	CategoryLabel  string                                 `json:"category_label,omitempty"`
	CoverageCount  int64                                  `json:"coverage_count"`
	Classification LedgerFeedRepresentativeClassification `json:"classification"`
	Summary        LedgerFeedRepresentativeSummary        `json:"summary"`
	Actors         *LedgerFeedRepresentativeActors        `json:"actors,omitempty"`
}

type LedgerFeedRepresentativeClassification struct {
	TxType     string `json:"tx_type,omitempty"`
	Subtype    string `json:"subtype,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

type LedgerFeedRepresentativeSummary struct {
	Description        string `json:"description,omitempty"`
	Amount             string `json:"amount,omitempty"`
	AmountDisplay      string `json:"amount_display,omitempty"`
	Asset              string `json:"asset,omitempty"`
	FunctionName       string `json:"function_name,omitempty"`
	ProtocolLabel      string `json:"protocol_label,omitempty"`
	ProtocolContractID string `json:"protocol_contract_id,omitempty"`
}

type LedgerFeedRepresentativeActors struct {
	Primary   *LedgerFeedRepresentativeActor `json:"primary,omitempty"`
	Secondary *LedgerFeedRepresentativeActor `json:"secondary,omitempty"`
}

type LedgerFeedRepresentativeActor struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Type  string `json:"type,omitempty"`
}

type LedgerFeedSummaryComposition struct {
	DominantTxType      string  `json:"dominant_tx_type,omitempty"`
	DominantTxTypeCount int64   `json:"dominant_tx_type_count"`
	SorobanSharePct     float64 `json:"soroban_share_pct"`
	FailedSharePct      float64 `json:"failed_share_pct"`
}

type LedgerFeedSummaryProvenance struct {
	ClassificationSource string `json:"classification_source,omitempty"`
	UtilizationSource    string `json:"utilization_source,omitempty"`
	SamplingSource       string `json:"sampling_source,omitempty"`
	Partial              bool   `json:"partial"`
}

// HomeSummaryResponse matches /home/summary.
type HomeSummaryResponse struct {
	Network                   string                         `json:"network"`
	GeneratedAt               string                         `json:"generated_at,omitempty"`
	Header                    HomeSummaryHeader              `json:"header"`
	Hero                      HomeSummaryHero                `json:"hero"`
	Alert                     HomeSummaryAlert               `json:"alert"`
	ContractsNeedingAttention []HomeSummaryAttentionContract `json:"contracts_needing_attention,omitempty"`
	Leaders                   []HomeSummaryLeader            `json:"leaders,omitempty"`
	Utilization               HomeSummaryUtilization         `json:"utilization"`
	Meta                      HomeSummaryMeta                `json:"meta"`
	Provenance                HomeSummaryProvenance          `json:"provenance"`
}

type HomeSummaryHeader struct {
	LatestLedgerSequence int64  `json:"latest_ledger_sequence"`
	LatestLedgerClosedAt string `json:"latest_ledger_closed_at,omitempty"`
}

type HomeSummaryHero struct {
	Health       HomeSummaryHeroHealth       `json:"health"`
	LatestLedger HomeSummaryHeroLatestLedger `json:"latest_ledger"`
	Cadence      HomeSummaryHeroCadence      `json:"cadence"`
	Contracts    HomeSummaryHeroContracts    `json:"contracts"`
	Soroban      HomeSummaryHeroSoroban      `json:"soroban"`
	Trends       HomeSummaryHeroTrends       `json:"trends"`
	TTL          HomeSummaryHeroTTL          `json:"ttl"`
	ActivityMix  HomeSummaryHeroActivityMix  `json:"activity_mix"`
}

type HomeSummaryHeroHealth struct {
	Status       string `json:"status,omitempty"`
	LoadBand     string `json:"load_band,omitempty"`
	ActivityBand string `json:"activity_band,omitempty"`
}

type HomeSummaryHeroLatestLedger struct {
	Sequence         int64  `json:"sequence"`
	ClosedAt         string `json:"closed_at,omitempty"`
	TransactionCount int64  `json:"transaction_count,omitempty"`
	OperationCount   int64  `json:"operation_count,omitempty"`
}

type HomeSummaryHeroCadence struct {
	AvgCloseSeconds       float64 `json:"avg_close_seconds,omitempty"`
	TxPerLedgerRecentAvg  int64   `json:"tx_per_ledger_recent_avg,omitempty"`
	OpsPerLedgerRecentAvg int64   `json:"ops_per_ledger_recent_avg,omitempty"`
}

type HomeSummaryHeroContracts struct {
	Active24h int64 `json:"active_24h,omitempty"`
}

type HomeSummaryHeroSoroban struct {
	InstructionPct float64 `json:"instruction_pct,omitempty"`
	ReadWritePct   float64 `json:"read_write_pct,omitempty"`
}

type HomeSummaryHeroTrends struct {
	TxVs24hAvgPct       float64 `json:"tx_vs_24h_avg_pct,omitempty"`
	AgentActivityWoWPct float64 `json:"agent_activity_wow_pct,omitempty"`
	AnomalyDetected     bool    `json:"anomaly_detected"`
}

type HomeSummaryHeroTTL struct {
	ExpiringContractCount int64 `json:"expiring_contract_count,omitempty"`
	WorstRemainingHours   int64 `json:"worst_remaining_hours,omitempty"`
	WorstRemainingLedgers int64 `json:"worst_remaining_ledgers,omitempty"`
}

type HomeSummaryHeroActivityMix struct {
	AgentTx24h        int64 `json:"agent_tx_24h,omitempty"`
	SwapTx24h         int64 `json:"swap_tx_24h,omitempty"`
	ContractCallTx24h int64 `json:"contract_call_tx_24h,omitempty"`
}

type HomeSummaryAlert struct {
	Type                  string   `json:"type,omitempty"`
	Severity              string   `json:"severity,omitempty"`
	AffectedContractCount int64    `json:"affected_contract_count,omitempty"`
	WorstRemainingHours   int64    `json:"worst_remaining_hours,omitempty"`
	TopContracts          []string `json:"top_contracts,omitempty"`
}

type HomeSummaryAttentionContract struct {
	ContractID       string  `json:"contract_id"`
	ProtocolName     string  `json:"protocol_name,omitempty"`
	ContractName     string  `json:"contract_name,omitempty"`
	Severity         string  `json:"severity,omitempty"`
	RemainingLedgers int64   `json:"remaining_ledgers,omitempty"`
	RemainingHours   int64   `json:"remaining_hours,omitempty"`
	RemainingHuman   string  `json:"remaining_human,omitempty"`
	RunwayPct        float64 `json:"runway_pct,omitempty"`
	Status           string  `json:"status,omitempty"`
}

type HomeSummaryLeader struct {
	ContractID       string   `json:"contract_id"`
	ProtocolName     string   `json:"protocol_name,omitempty"`
	ContractName     string   `json:"contract_name,omitempty"`
	CallCount24h     int64    `json:"call_count_24h,omitempty"`
	UniqueCallers24h int64    `json:"unique_callers_24h,omitempty"`
	DominantActions  []string `json:"dominant_actions,omitempty"`
	GrowthPct        float64  `json:"growth_pct,omitempty"`
}

type HomeSummaryUtilization struct {
	InstructionPct      float64 `json:"instruction_pct,omitempty"`
	InstructionUsed     int64   `json:"instruction_used,omitempty"`
	InstructionLimit    int64   `json:"instruction_limit,omitempty"`
	ReadWritePct        float64 `json:"read_write_pct,omitempty"`
	ReadWriteUsedBytes  int64   `json:"read_write_used_bytes,omitempty"`
	ReadWriteLimitBytes int64   `json:"read_write_limit_bytes,omitempty"`
	TxSizePct           float64 `json:"tx_size_pct,omitempty"`
	AvgTxSizeBytes      int64   `json:"avg_tx_size_bytes,omitempty"`
	TxSizeLimitBytes    int64   `json:"tx_size_limit_bytes,omitempty"`
	SourceLedger        int64   `json:"source_ledger,omitempty"`
}

type HomeSummaryMeta struct {
	LatestLedgerAgeSeconds int64 `json:"latest_ledger_age_seconds,omitempty"`
	LookupAvgMS            int64 `json:"lookup_avg_ms,omitempty"`
	LookupP99MS            int64 `json:"lookup_p99_ms,omitempty"`
	KnownProtocolCount     int64 `json:"known_protocol_count,omitempty"`
	HistoryStartProtocol   int64 `json:"history_start_protocol,omitempty"`
}

type HomeSummaryProvenance struct {
	Route         string   `json:"route,omitempty"`
	DataSource    string   `json:"data_source,omitempty"`
	Partial       bool     `json:"partial"`
	GeneratedFrom []string `json:"generated_from,omitempty"`
}

// --- Explorer Events ---

// ExplorerEventsResponse matches /explorer/events response.
type ExplorerEventsResponse struct {
	Meta       ExplorerEventMeta `json:"meta"`
	Events     []ExplorerEvent   `json:"events"`
	HasMore    bool              `json:"has_more"`
	NextCursor *string           `json:"next_cursor"`
	Count      int               `json:"count"`
}

type ExplorerEventMeta struct {
	MatchedCount    int                      `json:"matched_count"`
	CountCapped     bool                     `json:"count_capped"`
	LedgerRange     ExplorerEventLedgerRange `json:"ledger_range"`
	EventsPerSecond *float64                 `json:"events_per_second"`
}

type ExplorerEventLedgerRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type ExplorerEvent struct {
	EventID                  string  `json:"event_id"`
	Type                     string  `json:"type"`
	Protocol                 *string `json:"protocol"`
	ContractID               *string `json:"contract_id"`
	ContractName             *string `json:"contract_name"`
	ContractSymbol           *string `json:"contract_symbol"`
	LedgerSequence           int64   `json:"ledger_sequence"`
	TransactionHash          string  `json:"transaction_hash"`
	ClosedAt                 string  `json:"closed_at"`
	Successful               bool    `json:"successful"` // Deprecated API alias; use TransactionSuccessful for UI status.
	TransactionSuccessful    *bool   `json:"transaction_successful"`
	InSuccessfulContractCall *bool   `json:"in_successful_contract_call"`
	Topic0                   *string `json:"topic0"`
	Topic1                   *string `json:"topic1"`
	Topic2                   *string `json:"topic2"`
	Topic3                   *string `json:"topic3"`
	TopicsDecoded            *string `json:"topics_decoded"`
	Data                     *string `json:"data"`
	DataDecoded              *string `json:"data_decoded"`
	EventIndex               int     `json:"event_index"`
	OperationIndex           int     `json:"operation_index"`
}

// PublicSuccessful returns the transaction-scoped event status for explorer UI.
// It falls back to the deprecated successful field for compatibility with older Gateway responses.
func (e ExplorerEvent) PublicSuccessful() bool {
	if e.TransactionSuccessful != nil {
		return *e.TransactionSuccessful
	}
	return e.Successful
}

// --- Transaction Effects ---

// TransactionEffectsResponse matches /silver/effects/transaction/{hash} response.
type TransactionEffectsResponse struct {
	TransactionHash string              `json:"transaction_hash"`
	Count           int                 `json:"count"`
	Effects         []TransactionEffect `json:"effects"`
}

type TransactionEffect struct {
	LedgerSequence   int64          `json:"ledger_sequence"`
	TransactionHash  string         `json:"transaction_hash"`
	OperationIndex   int            `json:"operation_index"`
	EffectIndex      int            `json:"effect_index"`
	OperationID      int64          `json:"operation_id,omitempty"`
	EffectType       int            `json:"effect_type"`
	EffectTypeString string         `json:"effect_type_string"`
	AccountID        string         `json:"account_id"`
	Asset            *EffectAsset   `json:"asset,omitempty"`
	Amount           string         `json:"amount,omitempty"`
	Details          map[string]any `json:"details,omitempty"`
	Timestamp        string         `json:"timestamp"`
}

type EffectAsset struct {
	Code   string `json:"code"`
	Type   string `json:"type"`
	Issuer string `json:"issuer,omitempty"`
}

// ExplorerEventsParams holds query parameters for GetExplorerEvents.
type ExplorerEventsParams struct {
	Type         string
	Tab          string
	ContractID   string
	ContractName string
	TxHash       string
	TopicMatch   string
	StartLedger  int64
	EndLedger    int64
	Limit        int
	Cursor       string
	Order        string
}
