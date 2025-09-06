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
	Text  string
	Value int
	Min   int
	Max   int
}

// TradeQuery represents the JSON structure for PoE 2 trade API
type TradeQuery struct {
	Query struct {
		Status struct {
			Option string `json:"option"`
		} `json:"status"`
		Name    string `json:"name,omitempty"`
		Type    string `json:"type,omitempty"`
		Stats   []interface{} `json:"stats"`
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

	stat := &ItemStat{Text: cleanLine}

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
			}
		}
	}

	return stat
}

// buildAdvancedTradeSearchURL constructs a PoE 2 trade site URL with comprehensive search parameters
func (i *Input) buildAdvancedTradeSearchURL(item *ItemData) string {
	query := TradeQuery{}
	
	// Basic query setup
	query.Query.Status.Option = "online"
	query.Sort.Price = "asc"
	query.Query.Stats = make([]interface{}, 0)

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
