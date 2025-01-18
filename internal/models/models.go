package models

import (
	"regexp"
	"time"
)

// TradeEntry represents a trade-related log entry
type TradeEntry struct {
	Timestamp   time.Time
	TriggerType string
	PlayerName  string
	Message     string
}

// Trigger represents a log trigger with its compiled regular expression
type Trigger struct {
	Pattern string
	Regexp  *regexp.Regexp
}
