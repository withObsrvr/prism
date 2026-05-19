package search

import (
	"net/url"
	"regexp"
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
