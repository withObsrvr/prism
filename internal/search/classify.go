package search

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type ClassType string

const (
	ClassUnknown    ClassType = "unknown"
	ClassTxHash     ClassType = "transaction"
	ClassAccount    ClassType = "account"
	ClassContract   ClassType = "contract"
	ClassMuxed      ClassType = "muxed_account"
	ClassLedger     ClassType = "ledger"
	ClassFederation ClassType = "federation"
)

type Classification struct {
	Type  ClassType
	Value string
}

func (c Classification) Known() bool { return c.Type != "" && c.Type != ClassUnknown }

func (c Classification) URL() string {
	switch c.Type {
	case ClassTxHash:
		return "/v2/tx/" + c.Value
	case ClassAccount:
		return "/v2/account/" + c.Value
	case ClassContract:
		return "/v2/contract/" + c.Value
	case ClassLedger:
		return "/v2/ledger/" + c.Value
	case ClassMuxed:
		return "/v2/explore?account=" + url.QueryEscape(c.Value)
	case ClassFederation:
		return "/v2/explore?q=" + url.QueryEscape(c.Value)
	default:
		return ""
	}
}

var federationRE = regexp.MustCompile(`^[A-Za-z0-9._%+-]+\*[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)

var (
	embeddedTxRE         = regexp.MustCompile(`(?i)(^|[^0-9a-f])([0-9a-f]{64})([^0-9a-f]|$)`)
	embeddedStrKeyRE     = regexp.MustCompile(`(?i)(^|[^a-z2-7])([gc][a-z2-7]{55})([^a-z2-7]|$)`)
	embeddedMuxedRE      = regexp.MustCompile(`(?i)(^|[^a-z2-7])(m[a-z2-7]{55,69})([^a-z2-7]|$)`)
	embeddedLedgerRE     = regexp.MustCompile(`(?i)\bledger(?:\s+(?:number|sequence))?\s*#?\s*([1-9][0-9]*)\b`)
	embeddedFederationRE = regexp.MustCompile(`[A-Za-z0-9._%+-]+\*[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
)

func Classify(input string) Classification {
	q := strings.TrimSpace(input)
	if q == "" {
		return Classification{Type: ClassUnknown}
	}
	if len(q) == 64 && isHex(q) {
		return Classification{Type: ClassTxHash, Value: strings.ToLower(q)}
	}
	upper := strings.ToUpper(q)
	if len(upper) == 56 && isStrkeyBody(upper) {
		switch upper[0] {
		case 'G':
			return Classification{Type: ClassAccount, Value: upper}
		case 'C':
			return Classification{Type: ClassContract, Value: upper}
		}
	}
	if strings.HasPrefix(upper, "M") && len(upper) >= 56 && len(upper) <= 70 && isStrkeyBody(upper) {
		return Classification{Type: ClassMuxed, Value: upper}
	}
	if isDigits(q) {
		if n, err := strconv.ParseInt(q, 10, 64); err == nil && n > 0 {
			return Classification{Type: ClassLedger, Value: q}
		}
	}
	if federationRE.MatchString(q) {
		return Classification{Type: ClassFederation, Value: q}
	}
	return Classification{Type: ClassUnknown, Value: q}
}

// ExtractIdentifier finds a supported identifier inside surrounding prose.
// Exact identifiers continue to use Classify so both paths share canonical
// routing behavior.
func ExtractIdentifier(input string) Classification {
	if exact := Classify(input); exact.Known() {
		return exact
	}

	type candidate struct {
		start int
		value string
	}
	candidates := make([]candidate, 0, 5)
	appendMatch := func(re *regexp.Regexp, group int) {
		match := re.FindStringSubmatchIndex(input)
		groupStart := group * 2
		if match == nil || groupStart+1 >= len(match) || match[groupStart] < 0 {
			return
		}
		candidates = append(candidates, candidate{start: match[groupStart], value: input[match[groupStart]:match[groupStart+1]]})
	}
	appendMatch(embeddedTxRE, 2)
	appendMatch(embeddedStrKeyRE, 2)
	appendMatch(embeddedMuxedRE, 2)
	appendMatch(embeddedLedgerRE, 1)
	appendMatch(embeddedFederationRE, 0)
	if len(candidates) == 0 {
		return Classification{Type: ClassUnknown, Value: strings.TrimSpace(input)}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].start < candidates[j].start })
	return Classify(candidates[0].value)
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isStrkeyBody(s string) bool {
	for _, c := range s {
		if !(c >= 'A' && c <= 'Z' || c >= '2' && c <= '7') {
			return false
		}
	}
	return true
}
