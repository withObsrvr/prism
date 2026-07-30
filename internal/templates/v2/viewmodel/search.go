package viewmodelv2

import componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"

type SearchUnsupportedExample struct {
	Label string
	Href  string
}

type SearchUnsupportedData struct {
	Header         componentsv2.HeaderData
	Query          string
	Interpretation string
	RuleID         string
	Examples       []SearchUnsupportedExample
}
