package models

import (
	"regexp"
	"time"
)

// TradeEntry represents a trade-related log entry
type TradeEntry struct {
	Timestamp      time.Time
	TriggerType    string
	PlayerName     string
	League         string
	ItemName       string
	CurrencyAmount float64
	CurrencyType   string
	StashTab       string
	Position       struct {
		Left int
		Top  int
	}
	Message      string
	IsBuyRequest bool
}

// Trigger represents a log trigger with its compiled regular expression
type Trigger struct {
	Pattern string
	Regexp  *regexp.Regexp
}
