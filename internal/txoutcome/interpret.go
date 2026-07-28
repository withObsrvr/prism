package txoutcome

import (
	"fmt"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
)

// Interpretation is Prism's deterministic, presentation-neutral reading of a
// frozen transaction outcome packet. It contains no inferred facts beyond the
// packet and is shared by search answers and the transaction hero.
type Interpretation struct {
	Outcome               string
	Heading               string
	Summary               string
	ReasonCode            string
	ReasonLabel           string
	PhaseLabel            string
	OperationLabel        string
	OperationNumber       int
	FunctionName          string
	ContractID            string
	ArgumentLabels        []string
	RolledBackOperations  int
	NotExecutedOperations int
	EvidenceStatus        string
	ReasonAvailable       bool
	Caveats               []string
}

func Interpret(packet *gateway.TransactionOutcome) Interpretation {
	if packet == nil {
		return Interpretation{
			Outcome:        "unknown",
			Heading:        "Failure reason unavailable",
			Summary:        "Prism could not load the structured transaction outcome evidence.",
			EvidenceStatus: "unavailable",
		}
	}

	result := Interpretation{
		Outcome:        strings.ToLower(strings.TrimSpace(packet.Outcome)),
		EvidenceStatus: strings.ToLower(strings.TrimSpace(packet.Status)),
	}
	if result.EvidenceStatus == "" {
		result.EvidenceStatus = "unavailable"
	}
	for _, operation := range packet.Operations {
		if operation.ExecutionOutcome == "succeeded" && !operation.AppliedToLedger {
			result.RolledBackOperations++
		}
		if operation.ExecutionOutcome == "not_executed" {
			result.NotExecutedOperations++
		}
	}
	if invocation := packet.PrimaryInvocation; invocation != nil {
		result.FunctionName = strings.TrimSpace(invocation.FunctionName)
		result.ContractID = strings.TrimSpace(invocation.ContractID)
		for _, argument := range invocation.Arguments {
			label := strings.TrimSpace(argument.Display)
			if label == "" && argument.Value != nil {
				label = strings.TrimSpace(fmt.Sprint(argument.Value))
			}
			if label == "" {
				label = strings.TrimSpace(argument.Type)
			}
			if label != "" {
				result.ArgumentLabels = append(result.ArgumentLabels, label)
			}
		}
	}
	result.Caveats = caveatMessages(packet.Caveats)

	if result.Outcome == "succeeded" {
		result.Heading = "Transaction succeeded"
		result.Summary = "The authoritative transaction result confirms that its effects were applied to the ledger."
		result.ReasonCode = packet.TransactionResult.NormalizedCode
		result.ReasonLabel = "Success"
		result.ReasonAvailable = true
		return result
	}

	result.Heading = "Transaction failed"
	result.Summary = "The authoritative transaction result confirms that no transaction effects were applied to the ledger."
	failure := packet.Failure
	if failure == nil {
		result.Summary = "The transaction failed, but the evidence packet does not include a resolved failure reason."
		return result
	}

	result.ReasonCode = strings.TrimSpace(failure.NormalizedCode)
	result.ReasonLabel, result.Summary, result.ReasonAvailable = explainFailure(result.ReasonCode, failure.Status)
	result.PhaseLabel = phaseLabel(failure.Phase)
	result.OperationLabel = operationLabel(failure.OperationType)
	if failure.OperationIndex != nil {
		result.OperationNumber = *failure.OperationIndex + 1
	}

	switch {
	case result.FunctionName != "" && failure.Scope == "host_function":
		result.Heading = result.FunctionName + "() failed"
	case result.OperationLabel != "":
		result.Heading = result.OperationLabel + " failed"
	case failure.Phase == "transaction_validation" || failure.Phase == "soroban_validation":
		result.Heading = "Transaction rejected"
	}

	if result.RolledBackOperations > 0 {
		result.Summary += fmt.Sprintf(" %s executed successfully earlier, but %s not applied because the transaction failed.", operationCountLabel(result.RolledBackOperations), wasWere(result.RolledBackOperations))
	}
	if result.NotExecutedOperations > 0 {
		result.Summary += " " + notExecutedLabel(result.NotExecutedOperations) + " did not execute."
	}
	return result
}

func explainFailure(code, status string) (label, summary string, available bool) {
	if status != "ready" || code == "" || code == "unknown" || code == "transaction_failed" {
		return "Reason unresolved", "The transaction failed, but the available result evidence does not identify a more specific reason.", false
	}
	if known, ok := failureReasons[code]; ok {
		return known.label, known.summary, true
	}
	return humanCode(code), "The transaction returned result code " + code + ".", true
}

type failureReason struct {
	label   string
	summary string
}

var failureReasons = map[string]failureReason{
	"insufficient_fee":                             {"Insufficient fee", "The transaction was rejected because its fee was below the required amount."},
	"bad_sequence":                                 {"Invalid sequence", "The transaction was rejected because the source account sequence was not valid."},
	"sequence_precondition_failed":                 {"Sequence precondition failed", "The transaction did not satisfy its account-sequence precondition."},
	"time_bounds_not_satisfied":                    {"Time bounds not satisfied", "The transaction was submitted outside its permitted time bounds."},
	"invalid_authorization":                        {"Invalid authorization", "The transaction did not contain valid authorization for the requested action."},
	"insufficient_balance":                         {"Insufficient source balance", "The source account did not have enough balance to satisfy transaction-level requirements."},
	"source_account_missing":                       {"Source account missing", "The transaction source account was not present on the ledger."},
	"missing_operation":                            {"No operations", "The transaction was rejected because it did not contain an operation."},
	"malformed":                                    {"Malformed transaction", "The transaction envelope was malformed."},
	"soroban_invalid":                              {"Invalid Soroban transaction", "The transaction did not satisfy Soroban validation requirements."},
	"not_supported":                                {"Operation not supported", "The network did not support the requested transaction behavior."},
	"bad_sponsorship":                              {"Invalid sponsorship", "The transaction contained an invalid sponsorship relationship."},
	"internal_error":                               {"Network processing error", "The network reported an internal error while processing the transaction."},
	"payment_no_destination":                       {"Destination account missing", "The payment could not be applied because the destination account does not exist."},
	"payment_no_trust":                             {"Destination does not trust the asset", "The payment could not be applied because the destination has no trustline for the asset."},
	"payment_underfunded":                          {"Payment underfunded", "The payment source did not have enough available balance for the requested amount."},
	"payment_line_full":                            {"Destination trustline full", "The payment would have exceeded the destination trustline limit."},
	"payment_no_issuer":                            {"Asset issuer missing", "The payment asset issuer does not exist on the ledger."},
	"payment_not_authorized":                       {"Payment not authorized", "The destination is not authorized to hold the payment asset."},
	"create_account_underfunded":                   {"Account creation underfunded", "The source account did not have enough balance to create the destination account."},
	"create_account_low_reserve":                   {"Starting balance below reserve", "The requested starting balance was below the network minimum reserve."},
	"create_account_already_exist":                 {"Destination account already exists", "The account-creation operation targeted an account that already exists."},
	"invoke_host_function_trapped":                 {"Contract execution trapped", "The Soroban host stopped the contract invocation because execution trapped."},
	"invoke_host_function_resource_limit_exceeded": {"Resource limit exceeded", "The contract invocation exceeded an applicable Soroban resource limit."},
}

func phaseLabel(phase string) string {
	switch phase {
	case "transaction_validation":
		return "Transaction validation"
	case "soroban_validation":
		return "Soroban validation"
	case "transaction_execution":
		return "Transaction execution"
	case "operation_execution":
		return "Operation execution"
	case "soroban_host":
		return "Soroban host execution"
	case "resource_limits":
		return "Soroban resource limits"
	default:
		return humanCode(phase)
	}
}

func operationLabel(operationType string) string {
	value := strings.ToLower(strings.TrimSpace(operationType))
	switch value {
	case "payment":
		return "Payment"
	case "invoke_host_function":
		return "Contract invocation"
	case "create_account":
		return "Account creation"
	case "change_trust":
		return "Trustline change"
	case "path_payment_strict_receive", "path_payment_strict_send":
		return "Path payment"
	case "manage_sell_offer", "manage_buy_offer", "create_passive_sell_offer":
		return "Offer operation"
	default:
		return humanCode(value)
	}
}

func humanCode(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func caveatMessages(caveats []gateway.TransactionOutcomeCaveat) []string {
	seen := make(map[string]struct{}, len(caveats))
	out := make([]string, 0, len(caveats))
	for _, caveat := range caveats {
		message := strings.TrimSpace(caveat.Message)
		if message == "" {
			message = humanCode(caveat.Code)
		}
		if message == "" {
			continue
		}
		if _, ok := seen[message]; ok {
			continue
		}
		seen[message] = struct{}{}
		out = append(out, message)
	}
	return out
}

func operationCountLabel(count int) string {
	if count == 1 {
		return "One earlier operation"
	}
	return fmt.Sprintf("%d earlier operations", count)
}

func wasWere(count int) string {
	if count == 1 {
		return "was"
	}
	return "were"
}

func notExecutedLabel(count int) string {
	if count == 1 {
		return "One operation"
	}
	return fmt.Sprintf("%d operations", count)
}
