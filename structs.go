package main

type Suggestion struct {
	SuggType       string `json:"suggType"`
	Value          string `json:"value"`
	RefTag         string `json:"refTag"`
	StrategyId     string `json:"strategyId"`
	Ghost          bool   `json:"ghost"`
	Help           bool   `json:"help"`
	Fallback       bool   `json:"fallback"`
	SpellCorrected bool   `json:"spellCorrected"`
	BlackListed    bool   `json:"blackListed"`
	XcatOnly       bool   `json:"xcatOnly"`
}

type KeywordSuggestions struct {
	Alias             string       `json:"alias"`
	Prefix            string       `json:"prefix"`
	Suffix            string       `json:"suffix"`
	Suggestions       []Suggestion `json:"suggestions"`
	SuggestionTitleId string       `json:"suggestionTitleId"`
	ResponseId        string       `json:"responseId"`
	Shuffled          bool         `json:"shuffled"`
}

type Keyword struct {
	Keyword          string
	TotalResultCount int64
}

type Context struct {
	KeywordsFound int
	TLD           string
}
