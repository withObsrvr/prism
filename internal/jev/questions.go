package jev

func AskRouterQuestions() map[string]Question {
	return map[string]Question{
		"asks_about_failures": {
			Type:         "noul",
			Instructions: "Does `query` ask whether failures are occurring?",
			Criteria: map[string]string{
				"true":  "The user asks whether one or more transactions, swaps, invocations, or operations are failing.",
				"false": "The user does not ask whether failures are occurring.",
			},
		},
		"asks_about_activity": {
			Type:         "noul",
			Instructions: "Does `query` ask about activity volume, busyness, or quietness?",
			Criteria: map[string]string{
				"true":  "The user asks whether something is busy, active, quiet, frequently used, or has a particular activity level.",
				"false": "The user does not ask about activity volume.",
			},
		},
		"subject_kind": {
			Type:         "choice",
			Instructions: "What kind of subject is the user asking about in `query`? Classify named assets such as XLM, USDC, EURC, AQUA, and asset codes as asset—not protocol.",
			Criteria: map[string]string{
				"transaction": "One specific transaction or transaction hash.",
				"asset":       "A Stellar asset, token, or asset code such as XLM, USDC, EURC, or AQUA.",
				"protocol":    "A named application or protocol such as Soroswap, Blend, Phoenix, or Aquarius.",
				"contract":    "A specific Soroban contract or contract address.",
				"network":     "Stellar or Soroban activity generally, without a narrower subject.",
				"unknown":     "The subject cannot be determined.",
			},
		},
		"request_kind": {
			Type:         "choice",
			Instructions: "What does the user want to determine from `query`?",
			Criteria: map[string]string{
				"activity":            "How busy, active, quiet, popular, or frequently used the subject is.",
				"failure_reason":      "Why one particular transaction or invocation failed.",
				"recent_failures":     "Whether multiple recent transactions, swaps, operations, or contract invocations are failing.",
				"expiration_risk":     "Whether contract state has limited TTL, may expire, or may be archived.",
				"general_information": "General information that does not match the other request types.",
			},
		},
		"evidence_scope": {
			Type:         "choice",
			Instructions: "Does `query` concern one specific occurrence or an aggregate pattern across multiple occurrences?",
			Criteria: map[string]string{
				"single_occurrence": "One particular transaction, swap, invocation, or failure.",
				"aggregate_pattern": "A trend, rate, recent set, or repeated behavior across multiple occurrences.",
				"unclear":           "The scope cannot be determined.",
			},
		},
		"trend_kind": {
			Type:         "choice",
			Instructions: "What kind of temporal comparison does the user request in `query`?",
			Criteria: map[string]string{
				"current_level": "The user asks only about the current level or state.",
				"continuing":    "The user asks whether an earlier condition is still happening.",
				"recovery":      "The user asks whether an earlier negative condition has stopped or returned to normal.",
				"increase":      "The user asks whether activity or failures have increased.",
				"decrease":      "The user asks whether activity or failures have decreased.",
				"none":          "No temporal comparison is requested.",
			},
		},
		"time_window": {
			Type:         "choice",
			Instructions: "What explicit time period does the user request in `query`?",
			Criteria: map[string]string{
				"one_hour":         "The user explicitly requests the current or previous hour.",
				"one_day":          "The user explicitly requests today, the last 24 hours, or the past day.",
				"one_week":         "The user explicitly requests this week or the last seven days.",
				"one_month":        "The user explicitly requests this month or the last 30 days.",
				"current_snapshot": "The user asks about the state right now without specifying a historical duration.",
				"unspecified":      "No explicit time period is expressed.",
			},
		},
	}
}
