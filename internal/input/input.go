package input

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"

	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/logger"
	"hypr-exiled/pkg/notify"

	"hypr-exiled/internal/poe/window"
	"hypr-exiled/internal/wm"
)

type Input struct {
	windowManager *wm.Manager
	detector      *window.Detector
	log           *logger.Logger
	notifier      *notify.NotifyService
}

// Typing/timing parameters (tune as needed; consider moving to config later).
const (
	focusDelay       = 150 * time.Millisecond // after focusing the game window
	chatFocusDelay   = 100 * time.Millisecond // after opening chat
	clearSelectDelay = 30 * time.Millisecond  // after Ctrl+A
	clearDeleteDelay = 30 * time.Millisecond  // after Backspace
	afterTypeDelay   = 40 * time.Millisecond  // after typing the command
	sendCooldown     = 120 * time.Millisecond // between consecutive commands

	typeCharDelayMs = 10 // per-character typing delay for robotgo.TypeStrDelay
)

func NewInput(detector *window.Detector) (*Input, error) {
	log := global.GetLogger()
	notifier := global.GetNotifier()

	return &Input{
		windowManager: detector.GetCurrentWm(),
		detector:      detector,
		log:           log,
		notifier:      notifier,
	}, nil
}

func (i *Input) ExecutePoECommands(commands []string) error {
	cfg := global.GetConfig()

	if !i.detector.IsActive() {
		return fmt.Errorf("%s needs to be running", cfg.GameNameByAppID(i.detector.ActiveAppID()))
	}

	window := i.detector.GetCurrentWindow()
	if err := i.windowManager.FocusWindow(window); err != nil {
		return fmt.Errorf("failed to focus window: %w", err)
	}

	// Decide profile: PoE1 = slow, PoE2 = fast
	slowTyping := i.isSlowTypingApp()

	if slowTyping {
		// Give PoE1 a moment to accept input after focusing the window.
		time.Sleep(focusDelay)
	}

	for _, cmd := range commands {
		i.log.Debug("Executing PoE command", "command", cmd, "window_class", window.Class)

		if slowTyping {
			// --- SLOW PROFILE (PoE1) ---
			robotgo.KeyTap("enter")     // open chat
			time.Sleep(chatFocusDelay)  // allow input to focus
			robotgo.KeyTap("a", "ctrl") // clear any stale input
			time.Sleep(clearSelectDelay)
			robotgo.KeyTap("backspace")
			time.Sleep(clearDeleteDelay)

			// Type with delay to avoid dropped characters in PoE1.
			robotgo.TypeStrDelay(cmd, typeCharDelayMs)
			time.Sleep(afterTypeDelay)

			robotgo.KeyTap("enter")  // send
			time.Sleep(sendCooldown) // small cooldown between commands
		} else {
			// --- FAST PROFILE (PoE2) ---
			robotgo.KeyTap("enter")
			robotgo.TypeStr(cmd)
			robotgo.KeyTap("enter")
			// No extra sleeps for PoE2
		}
	}
	return nil
}

func (i *Input) ExecuteHideout() error {
	return i.ExecutePoECommands([]string{"/hideout"})
}

func (i *Input) ExecuteKingsmarch() error {
	return i.ExecutePoECommands([]string{"/kingsmarch"})
}

// isSlowTypingApp decides if we should use the slow typing profile.
// Default: PoE1 → slow; PoE2 → fast.
// This avoids magic numbers by resolving via configured game names.
func (i *Input) isSlowTypingApp() bool {
	cfg := global.GetConfig()
	name := cfg.GameNameByAppID(i.detector.ActiveAppID())
	return name == "Path of Exile" // PoE1
}

// ExecuteSearch extracts item text from clipboard, parses it, and opens PoE 2 trade site
func (i *Input) ExecuteSearch() error {
	cfg := global.GetConfig()

	if !i.detector.IsActive() {
		return fmt.Errorf("%s needs to be running", cfg.GameNameByAppID(i.detector.ActiveAppID()))
	}

	// Focus the PoE window first
	window := i.detector.GetCurrentWindow()
	if err := i.windowManager.FocusWindow(window); err != nil {
		return fmt.Errorf("failed to focus window: %w", err)
	}

	// Give the window focus time
	time.Sleep(100 * time.Millisecond)

	// Copy item to clipboard (Ctrl+C)
	i.log.Debug("Copying item to clipboard")
	robotgo.KeyTap("c", "ctrl")
	
	// Wait for clipboard to be populated
	time.Sleep(200 * time.Millisecond)

	// Get clipboard content
	clipboardText, err := robotgo.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read clipboard: %w", err)
	}
	if clipboardText == "" {
		return fmt.Errorf("no item text found in clipboard")
	}

	i.log.Debug("Extracted item text", "text", clipboardText)

	// Parse the full item data
	itemData, err := i.parseItemData(clipboardText)
	if err != nil {
		return fmt.Errorf("failed to parse item data: %w", err)
	}

	i.log.Debug("Parsed item data", "item", itemData)

	// Construct PoE 2 trade site URL with full search parameters
	tradeURL := i.buildAdvancedTradeSearchURL(itemData)
	i.log.Debug("Opening trade URL", "url", tradeURL)

	// Open URL in default browser
	if err := i.openURL(tradeURL); err != nil {
		return fmt.Errorf("failed to open trade URL: %w", err)
	}

	return nil
}

// ItemData represents the parsed item information
type ItemData struct {
	Name        string
	BaseType    string
	Rarity      string
	ItemLevel   int
	Quality     int
	Corrupted   bool
	Sockets     int
	Properties  map[string]string
	Requirements map[string]int
	Stats       []ItemStat
	League      string
}

// ItemStat represents a modifier/stat on an item
type ItemStat struct {
	Text         string
	Value        int
	Min          int
	Max          int
	ModifierType string // "prefix", "suffix", "implicit", "unknown"
	StatID       string // Standardized stat identifier for trade API
	IsRange      bool   // Whether this stat represents a range vs exact value
}

// StatFilter represents a single stat filter in the trade query
type StatFilter struct {
	ID       string  `json:"id"`
	Value    *struct {
		Min *int `json:"min,omitempty"`
		Max *int `json:"max,omitempty"`
	} `json:"value,omitempty"`
	Disabled bool `json:"disabled,omitempty"`
}

// StatGroup represents a group of stat filters 
type StatGroup struct {
	Type    string       `json:"type"`    // "and", "or", "not"
	Filters []StatFilter `json:"filters"`
}

// TradeQuery represents the JSON structure for PoE 2 trade API
type TradeQuery struct {
	Query struct {
		Status struct {
			Option string `json:"option"`
		} `json:"status"`
		Name    string      `json:"name,omitempty"`
		Type    string      `json:"type,omitempty"`
		Stats   []StatGroup `json:"stats"`
		Filters struct {
			TypeFilters *struct {
				Filters struct {
					Quality *struct {
						Min int `json:"min,omitempty"`
					} `json:"quality,omitempty"`
					ItemLevel *struct {
						Min int `json:"min,omitempty"`
						Max int `json:"max,omitempty"`
					} `json:"ilvl,omitempty"`
				} `json:"filters"`
			} `json:"type_filters,omitempty"`
			MiscFilters *struct {
				Filters struct {
					Corrupted *struct {
						Option string `json:"option,omitempty"`
					} `json:"corrupted,omitempty"`
				} `json:"filters"`
			} `json:"misc_filters,omitempty"`
		} `json:"filters"`
	} `json:"query"`
	Sort struct {
		Price string `json:"price"`
	} `json:"sort"`
}

// parseItemData extracts comprehensive item information from tooltip text
func (i *Input) parseItemData(clipboardText string) (*ItemData, error) {
	lines := strings.Split(clipboardText, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty clipboard text")
	}

	item := &ItemData{
		Properties:   make(map[string]string),
		Requirements: make(map[string]int),
		Stats:        make([]ItemStat, 0),
		League:       "Standard", // Default league, could be configured
	}

	// Remove quantity prefix if present in first line
	firstLine := strings.TrimSpace(lines[0])
	quantityRegex := regexp.MustCompile(`^\d+x\s+`)
	firstLine = quantityRegex.ReplaceAllString(firstLine, "")

	// Parse PoE 2 format: first few lines contain item class, rarity, name, base type
	lineIndex := 0
	
	// Skip "Item Class:" line if present
	if strings.HasPrefix(firstLine, "Item Class:") {
		lineIndex = 1
	}

	// Parse rarity line
	if lineIndex < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[lineIndex]), "Rarity: ") {
		rarityLine := strings.TrimSpace(lines[lineIndex])
		item.Rarity = strings.ToLower(strings.TrimPrefix(rarityLine, "Rarity: "))
		lineIndex++
		
		// Get item name from next line
		if lineIndex < len(lines) {
			item.Name = strings.TrimSpace(lines[lineIndex])
			lineIndex++
		}
		
		// Get base type from next line
		if lineIndex < len(lines) {
			nextLine := strings.TrimSpace(lines[lineIndex])
			if nextLine != "" && !strings.HasPrefix(nextLine, "--------") {
				item.BaseType = nextLine
				lineIndex++
			}
		}
	} else {
		// If no rarity prefix, assume current line is the item name
		currentLine := firstLine
		if lineIndex < len(lines) {
			currentLine = strings.TrimSpace(lines[lineIndex])
		}
		item.Name = currentLine
		item.Rarity = "normal"
		lineIndex++
		
		// Try to get base type from next line if available
		if lineIndex < len(lines) {
			nextLine := strings.TrimSpace(lines[lineIndex])
			if nextLine != "" && !strings.HasPrefix(nextLine, "--------") {
				item.BaseType = nextLine
				lineIndex++
			}
		}
	}

	// Parse remaining lines for properties, requirements, and stats
	inSection := ""
	for idx := lineIndex; idx < len(lines); idx++ {
		line := strings.TrimSpace(lines[idx])
		if line == "" || line == "--------" {
			inSection = ""
			continue
		}

		// Detect sections and parse data
		if strings.Contains(line, "Item Level:") {
			if match := regexp.MustCompile(`Item Level: (\d+)`).FindStringSubmatch(line); match != nil {
				item.ItemLevel, _ = strconv.Atoi(match[1])
			}
		} else if strings.Contains(line, "Quality:") {
			if match := regexp.MustCompile(`Quality: \+(\d+)%`).FindStringSubmatch(line); match != nil {
				item.Quality, _ = strconv.Atoi(match[1])
			}
		} else if strings.Contains(line, "Corrupted") {
			item.Corrupted = true
		} else if strings.Contains(line, "Sockets:") {
			// Count socket characters (simple implementation)
			sockets := strings.Count(line, "R") + strings.Count(line, "G") + strings.Count(line, "B") + strings.Count(line, "W")
			item.Sockets = sockets
		} else if strings.HasPrefix(line, "Requirements:") {
			inSection = "requirements"
		} else if strings.HasPrefix(line, "Requires:") {
			// Parse requirements directly from this line
			reqRegex := regexp.MustCompile(`(Level|Str|Dex|Int) (\d+)`)
			matches := reqRegex.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				value, _ := strconv.Atoi(match[2])
				item.Requirements[strings.ToLower(match[1])] = value
			}
		} else if inSection == "requirements" && (strings.Contains(line, "Level ") || strings.Contains(line, "Str ") || strings.Contains(line, "Dex ") || strings.Contains(line, "Int ")) {
			// Parse requirements
			reqRegex := regexp.MustCompile(`(Level|Str|Dex|Int) (\d+)`)
			matches := reqRegex.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				value, _ := strconv.Atoi(match[2])
				item.Requirements[strings.ToLower(match[1])] = value
			}
		} else {
			// Try to parse as a stat/modifier
			stat := i.parseStatLine(line)
			if stat != nil {
				item.Stats = append(item.Stats, *stat)
			}
		}
	}

	return item, nil
}

// parseStatLine attempts to extract stat information from a line
func (i *Input) parseStatLine(line string) *ItemStat {
	// Remove color codes and extra formatting
	cleanLine := regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(line, "")
	cleanLine = strings.TrimSpace(cleanLine)

	if cleanLine == "" {
		return nil
	}

	// Skip lines that are not modifiers/stats
	skipPrefixes := []string{
		"Item Class:",
		"Rarity:",
		"Requires:",
		"Requirements:",
		"Item Level:",
		"Quality:",
		"Sockets:",
		"Grants Skill:",
		"--------",
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(cleanLine, prefix) {
			return nil
		}
	}

	// Skip short non-meaningful lines
	if len(cleanLine) <= 3 {
		return nil
	}

	// Try to extract numeric values from the line
	numberRegex := regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)
	numbers := numberRegex.FindAllString(cleanLine, -1)

	stat := &ItemStat{
		Text:         cleanLine,
		ModifierType: "unknown",
		IsRange:      false,
	}

	// Classify modifier type and extract stat ID
	i.classifyModifier(stat)

	if len(numbers) > 0 {
		// Parse the first number as the main value
		if val, err := strconv.Atoi(numbers[0]); err == nil {
			stat.Value = val
			stat.Min = val
			stat.Max = val
		}

		// If there are two numbers, treat them as min/max range
		if len(numbers) >= 2 {
			if min, err := strconv.Atoi(numbers[0]); err == nil {
				stat.Min = min
			}
			if max, err := strconv.Atoi(numbers[1]); err == nil {
				stat.Max = max
				stat.Value = (stat.Min + stat.Max) / 2 // Use average as main value
				stat.IsRange = true
			}
		}
	}

	return stat
}

// ModifierPattern represents a pattern for matching and classifying modifiers
type ModifierPattern struct {
	Pattern      *regexp.Regexp
	StatID       string
	ModifierType string
	Description  string
}

// classifyModifier attempts to classify a modifier and assign a stat ID based on common PoE 2 patterns
func (i *Input) classifyModifier(stat *ItemStat) {
	// Using hash-based stat IDs from Exiled-Exchange-2 data for PoE 2 trade API compatibility
	
	// Define patterns with correct hash-based stat IDs from Exiled-Exchange-2
	patterns := []ModifierPattern{
		// Spell Damage (Prefix) - explicit.stat_2974417149
		{regexp.MustCompile(`(\d+)% increased Spell Damage`), "explicit.stat_2974417149", "prefix", "Increased Spell Damage"},
		
		// Chaos Damage (Prefix) - explicit.stat_736967255
		{regexp.MustCompile(`(\d+)% increased Chaos Damage`), "explicit.stat_736967255", "prefix", "Increased Chaos Damage"},
		
		// Mana (Prefix) - explicit.stat_1050105434
		{regexp.MustCompile(`\+(\d+) to maximum Mana`), "explicit.stat_1050105434", "prefix", "Maximum Mana"},
		
		// Intelligence (Suffix) - explicit.stat_328541901
		{regexp.MustCompile(`\+(\d+) to Intelligence`), "explicit.stat_328541901", "suffix", "Intelligence"},
		
		// Spell Skills Level (Prefix) - explicit.stat_124131830
		{regexp.MustCompile(`\+(\d+) to Level of all Spell Skills`), "explicit.stat_124131830", "prefix", "Spell Skills Level"},
		
		// Mana on Kill (Suffix) - explicit.stat_1368271171
		{regexp.MustCompile(`Gain (\d+) Mana per Enemy Killed`), "explicit.stat_1368271171", "suffix", "Mana on Kill"},
		
		// More patterns with actual stat IDs can be added as needed
		// For now, focusing on the modifiers from your example
		
		// Life (Prefix) - Need to look up the actual ID
		{regexp.MustCompile(`\+(\d+) to maximum Life`), "", "prefix", "Maximum Life"},
		
		// Resistances (Suffix) - Need to look up the actual IDs
		{regexp.MustCompile(`\+(\d+)% to .* Resistance`), "", "suffix", "Resistance"},
		
		// Physical Damage (Prefix) - Need to look up the actual ID
		{regexp.MustCompile(`Adds (\d+)(?:-(\d+))? Physical Damage`), "", "prefix", "Added Physical Damage"},
		
		// Attack/Cast Speed (Suffix) - Need to look up the actual IDs
		{regexp.MustCompile(`(\d+)% increased Attack Speed`), "", "suffix", "Attack Speed"},
		{regexp.MustCompile(`(\d+)% increased Cast Speed`), "", "suffix", "Cast Speed"},
		
		// Critical Strike (Suffix) - Need to look up the actual IDs
		{regexp.MustCompile(`(\d+)% increased Critical Strike Chance`), "", "suffix", "Critical Strike Chance"},
		{regexp.MustCompile(`\+(\d+)% to Critical Strike Multiplier`), "", "suffix", "Critical Strike Multiplier"},
		
		// Accuracy (Suffix) - Need to look up the actual ID
		{regexp.MustCompile(`\+(\d+) to Accuracy Rating`), "", "suffix", "Accuracy Rating"},
		
		// Energy Shield (Prefix) - Need to look up the actual IDs
		{regexp.MustCompile(`\+(\d+) to maximum Energy Shield`), "", "prefix", "Maximum Energy Shield"},
		{regexp.MustCompile(`(\d+)% increased Energy Shield`), "", "prefix", "Increased Energy Shield"},
		
		// Armor/Evasion (Prefix) - Need to look up the actual IDs
		{regexp.MustCompile(`(\d+)% increased Armour`), "", "prefix", "Increased Armour"},
		{regexp.MustCompile(`(\d+)% increased Evasion Rating`), "", "prefix", "Increased Evasion"},
		
		// Movement Speed (Suffix) - Need to look up the actual ID
		{regexp.MustCompile(`(\d+)% increased Movement Speed`), "", "suffix", "Movement Speed"},
		
		// Attributes (Suffix) - Need to look up the actual IDs
		{regexp.MustCompile(`\+(\d+) to Strength`), "", "suffix", "Strength"},
		{regexp.MustCompile(`\+(\d+) to Dexterity`), "", "suffix", "Dexterity"},
		{regexp.MustCompile(`\+(\d+) to all Attributes`), "", "suffix", "All Attributes"},
	}

	// Try to match against known patterns
	for _, pattern := range patterns {
		if pattern.Pattern.MatchString(stat.Text) {
			stat.StatID = pattern.StatID
			stat.ModifierType = pattern.ModifierType
			i.log.Debug("Classified modifier", "text", stat.Text, "type", stat.ModifierType, "description", pattern.Description)
			return
		}
	}

	// If no pattern matched, try to make a best guess based on common conventions
	i.classifyByConvention(stat)
}

// classifyByConvention makes a best guess classification when exact patterns don't match
func (i *Input) classifyByConvention(stat *ItemStat) {
	text := strings.ToLower(stat.Text)
	
	// Common prefix indicators (usually defensive or damage stats)
	prefixIndicators := []string{
		"increased", "adds", "maximum life", "maximum mana", "maximum energy shield", 
		"spell damage", "physical damage", "chaos damage", "fire damage", "cold damage", "lightning damage",
		"level of", "grants skill",
	}
	
	// Common suffix indicators (usually utility stats)
	suffixIndicators := []string{
		"resistance", "accuracy", "critical strike", "attack speed", "cast speed", "movement speed",
		"intelligence", "strength", "dexterity", "per enemy killed", "on kill",
	}
	
	// Check for prefix patterns
	for _, indicator := range prefixIndicators {
		if strings.Contains(text, indicator) {
			stat.ModifierType = "prefix"
			stat.StatID = fmt.Sprintf("explicit.stat_%s", strings.ReplaceAll(indicator, " ", "_"))
			i.log.Debug("Classified as prefix by convention", "text", stat.Text, "indicator", indicator)
			return
		}
	}
	
	// Check for suffix patterns  
	for _, indicator := range suffixIndicators {
		if strings.Contains(text, indicator) {
			stat.ModifierType = "suffix" 
			stat.StatID = fmt.Sprintf("explicit.stat_%s", strings.ReplaceAll(indicator, " ", "_"))
			i.log.Debug("Classified as suffix by convention", "text", stat.Text, "indicator", indicator)
			return
		}
	}
	
	// Default to unknown
	i.log.Debug("Could not classify modifier", "text", stat.Text)
	stat.ModifierType = "unknown"
	stat.StatID = ""
}

// buildStatFilters converts ItemStats to StatFilters for the trade query
func (i *Input) buildStatFilters(stats []ItemStat) []StatFilter {
	var filters []StatFilter
	var classifiedCount = 0
	
	for _, stat := range stats {
		// Count classified stats for logging
		if stat.ModifierType != "unknown" {
			classifiedCount++
		}
		
		// Skip stats that don't have a stat ID (since we don't have the PoE 2 hash-based IDs)
		if stat.StatID == "" {
			i.log.Debug("Skipping stat without API ID", "text", stat.Text, "type", stat.ModifierType)
			continue
		}
		
		filter := StatFilter{
			ID:       stat.StatID,
			Disabled: false,
		}
		
		// Add value constraints based on the stat values
		if stat.Value > 0 {
			filter.Value = &struct {
				Min *int `json:"min,omitempty"`
				Max *int `json:"max,omitempty"`
			}{}
			
			// For exact values or small ranges, use tight constraints
			if !stat.IsRange || (stat.Max - stat.Min) <= 2 {
				// Use 80% of the stat value as minimum to find similar items
				minValue := int(float64(stat.Value) * 0.8)
				filter.Value.Min = &minValue
				i.log.Debug("Added exact stat filter", "id", stat.StatID, "min", minValue, "text", stat.Text)
			} else {
				// For ranges, use the minimum value as the search constraint
				minValue := stat.Min
				filter.Value.Min = &minValue
				i.log.Debug("Added range stat filter", "id", stat.StatID, "min", minValue, "text", stat.Text)
			}
		}
		
		filters = append(filters, filter)
	}
	
	// Log the classification results
	if classifiedCount > 0 {
		i.log.Info("Classified modifiers for search", 
			"total", len(stats), 
			"classified", classifiedCount, 
			"with_api_ids", len(filters))
		if len(filters) == 0 {
			i.log.Info("No stat filters added", "reason", "Some modifiers don't have hash-based stat IDs yet")
		}
	}
	
	return filters
}

// buildAdvancedTradeSearchURL constructs a PoE 2 trade site URL with comprehensive search parameters
func (i *Input) buildAdvancedTradeSearchURL(item *ItemData) string {
	query := TradeQuery{}
	
	// Basic query setup
	query.Query.Status.Option = "online"
	query.Sort.Price = "asc"
	query.Query.Stats = make([]StatGroup, 0)

	// Set item name/type
	if item.Name != "" {
		if item.Rarity == "unique" {
			query.Query.Name = item.Name
		} else if item.BaseType != "" {
			query.Query.Type = item.BaseType
		} else {
			query.Query.Type = item.Name
		}
	}

	// Add filters
	if item.Quality > 0 {
		if query.Query.Filters.TypeFilters == nil {
			query.Query.Filters.TypeFilters = &struct {
				Filters struct {
					Quality *struct {
						Min int `json:"min,omitempty"`
					} `json:"quality,omitempty"`
					ItemLevel *struct {
						Min int `json:"min,omitempty"`
						Max int `json:"max,omitempty"`
					} `json:"ilvl,omitempty"`
				} `json:"filters"`
			}{}
		}
		query.Query.Filters.TypeFilters.Filters.Quality = &struct {
			Min int `json:"min,omitempty"`
		}{Min: item.Quality}
	}

	if item.ItemLevel > 0 {
		if query.Query.Filters.TypeFilters == nil {
			query.Query.Filters.TypeFilters = &struct {
				Filters struct {
					Quality *struct {
						Min int `json:"min,omitempty"`
					} `json:"quality,omitempty"`
					ItemLevel *struct {
						Min int `json:"min,omitempty"`
						Max int `json:"max,omitempty"`
					} `json:"ilvl,omitempty"`
				} `json:"filters"`
			}{}
		}
		// Allow some range around the item level
		minLevel := item.ItemLevel - 5
		maxLevel := item.ItemLevel + 5
		if minLevel < 1 {
			minLevel = 1
		}
		query.Query.Filters.TypeFilters.Filters.ItemLevel = &struct {
			Min int `json:"min,omitempty"`
			Max int `json:"max,omitempty"`
		}{Min: minLevel, Max: maxLevel}
	}

	if item.Corrupted {
		if query.Query.Filters.MiscFilters == nil {
			query.Query.Filters.MiscFilters = &struct {
				Filters struct {
					Corrupted *struct {
						Option string `json:"option,omitempty"`
					} `json:"corrupted,omitempty"`
				} `json:"filters"`
			}{}
		}
		query.Query.Filters.MiscFilters.Filters.Corrupted = &struct {
			Option string `json:"option,omitempty"`
		}{Option: "true"}
	}

	// Add stat filters from parsed modifiers
	statFilters := i.buildStatFilters(item.Stats)
	if len(statFilters) > 0 {
		statGroup := StatGroup{
			Type:    "and",
			Filters: statFilters,
		}
		query.Query.Stats = append(query.Query.Stats, statGroup)
		i.log.Info("Added stat filters to search", "count", len(statFilters))
	} else if len(item.Stats) > 0 {
		i.log.Info("Using basic search", 
			"parsed_stats", len(item.Stats),
			"reason", "No modifiers matched known stat IDs")
	}

	// Serialize the query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		i.log.Error("Failed to marshal trade query", err)
		// Fallback to simple search
		return i.buildSimpleTradeSearchURL(item.Name)
	}

	// Construct the final URL
	baseURL := fmt.Sprintf("https://www.pathofexile.com/trade2/search/poe2/%s", item.League)
	encodedQuery := url.QueryEscape(string(queryJSON))
	
	return fmt.Sprintf("%s?q=%s", baseURL, encodedQuery)
}

// buildSimpleTradeSearchURL constructs a simple PoE 2 trade site URL as fallback
func (i *Input) buildSimpleTradeSearchURL(itemName string) string {
	baseURL := "https://www.pathofexile.com/trade2/search/poe2/Standard"
	
	// Create a simple query structure
	simpleQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"status": map[string]string{"option": "online"},
			"type":   itemName,
			"stats":  []interface{}{map[string]interface{}{"type": "and", "filters": []interface{}{}}},
		},
		"sort": map[string]string{"price": "asc"},
	}

	queryJSON, _ := json.Marshal(simpleQuery)
	encodedQuery := url.QueryEscape(string(queryJSON))
	
	return fmt.Sprintf("%s?q=%s", baseURL, encodedQuery)
}

// openURL opens the given URL in the default browser
func (i *Input) openURL(url string) error {
	var cmd *exec.Cmd
	
	// Determine the appropriate command based on the operating system
	// Since this is primarily for Linux (based on the project focus), use xdg-open
	cmd = exec.Command("xdg-open", url)
	
	return cmd.Start()
}
