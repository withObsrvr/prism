package humanize

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed rules/tx_functions.json
var txFunctionRulesJSON []byte

type txFunctionRuleSet struct {
	ProjectAliases map[string][]string              `json:"project_aliases,omitempty"`
	Functions      map[string]FunctionNarrationRule `json:"functions"`
	Overrides      []FunctionNarrationOverride      `json:"overrides,omitempty"`
}

type FunctionNarrationRule struct {
	Phrase      string      `json:"phrase"`
	Description string      `json:"description,omitempty"`
	Category    string      `json:"category,omitempty"`
	Signal      *SignalRule `json:"signal,omitempty"`
}

type FunctionNarrationOverride struct {
	Match       FunctionNarrationMatch `json:"match"`
	Phrase      string                 `json:"phrase"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Signal      *SignalRule            `json:"signal,omitempty"`
}

type FunctionNarrationMatch struct {
	FunctionName string `json:"function_name,omitempty"`
	ContractID   string `json:"contract_id,omitempty"`
	ContractName string `json:"contract_name,omitempty"`
	Project      string `json:"project,omitempty"`
}

type SignalRule struct {
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
}

var loadedTxFunctionRules = mustLoadTxFunctionRules()

func mustLoadTxFunctionRules() txFunctionRuleSet {
	var rules txFunctionRuleSet
	if err := json.Unmarshal(txFunctionRulesJSON, &rules); err != nil {
		panic(fmt.Errorf("humanize: parsing tx function rules: %w", err))
	}
	if rules.Functions == nil {
		rules.Functions = map[string]FunctionNarrationRule{}
	}
	return rules
}

func LookupFunctionNarration(functionName string) (FunctionNarrationRule, bool) {
	return LookupFunctionNarrationWithContext(functionName, "", "", "")
}

func LookupFunctionNarrationWithContext(functionName, contractID, contractName, project string) (FunctionNarrationRule, bool) {
	functionName = strings.TrimSpace(strings.ToLower(functionName))
	contractID = strings.TrimSpace(strings.ToUpper(contractID))
	contractName = strings.TrimSpace(strings.ToLower(contractName))
	project = strings.TrimSpace(strings.ToLower(project))
	if functionName == "" {
		return FunctionNarrationRule{}, false
	}

	for _, o := range loadedTxFunctionRules.Overrides {
		if o.Match.FunctionName != "" && strings.ToLower(o.Match.FunctionName) != functionName {
			continue
		}
		if o.Match.ContractID != "" && strings.ToUpper(o.Match.ContractID) != contractID {
			continue
		}
		if o.Match.ContractName != "" && strings.ToLower(o.Match.ContractName) != contractName {
			continue
		}
		if o.Match.Project != "" && strings.ToLower(o.Match.Project) != project {
			continue
		}
		return FunctionNarrationRule{Phrase: o.Phrase, Description: o.Description, Category: o.Category, Signal: o.Signal}, true
	}

	r, ok := loadedTxFunctionRules.Functions[functionName]
	return r, ok
}

func InferProject(contractName string) string {
	name := strings.ToLower(strings.TrimSpace(contractName))
	if name == "" {
		return ""
	}
	for project, aliases := range loadedTxFunctionRules.ProjectAliases {
		for _, alias := range aliases {
			if alias == "" {
				continue
			}
			if strings.Contains(name, strings.ToLower(alias)) {
				return project
			}
		}
	}
	return ""
}

func HumanizeFunctionName(functionName string) string {
	functionName = strings.TrimSpace(functionName)
	if functionName == "" {
		return ""
	}
	functionName = strings.ReplaceAll(functionName, "_", " ")
	functionName = strings.ReplaceAll(functionName, "-", " ")
	functionName = strings.Join(strings.Fields(functionName), " ")
	return functionName
}
