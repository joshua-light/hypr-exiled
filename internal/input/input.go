package input

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "math"
    "net/http"
    "net/url"
    "os"
    "os/exec"
    "regexp"
    "sort"
    "strconv"
    "strings"
	"time"

	"github.com/go-vgo/robotgo"

	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/logger"
	"hypr-exiled/pkg/notify"

	"hypr-exiled/internal/poe/window"
	"hypr-exiled/internal/wm"

	"hypr-exiled/internal/input/statsmap"
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

    // Ensure external stat mapping is loaded before parsing/classifying modifiers
    // so that classifyModifier can resolve hashed stat IDs.
    statsmap.Load()

    // Parse the full item data
    itemData, err := i.parseItemData(clipboardText)
	if err != nil {
		return fmt.Errorf("failed to parse item data: %w", err)
	}

	i.log.Debug("Parsed item data", "item", itemData)

	// Construct PoE 2 trade site URL with full search parameters
	// Initialize external stats mapping if available (from Exiled-Exchange-2)
	statsmap.Load()
	tradeURL := i.buildAdvancedTradeSearchURL(itemData)
	i.log.Debug("Opening trade URL", "url", tradeURL)

	// Open URL in default browser
	if err := i.openURL(tradeURL); err != nil {
		return fmt.Errorf("failed to open trade URL: %w", err)
	}

	return nil
}

// ExecutePrice extracts item text from clipboard, makes API requests to get pricing data
func (i *Input) ExecutePrice() (map[string]interface{}, error) {
    cfg := global.GetConfig()

	i.log.Debug("ExecutePrice called")
	if !i.detector.IsActive() {
		activeApp := cfg.GameNameByAppID(i.detector.ActiveAppID())
		i.log.Debug("PoE not active", "current_app", activeApp)
		return nil, fmt.Errorf("%s needs to be running", activeApp)
	}

	// Focus the PoE window first
	window := i.detector.GetCurrentWindow()
	if err := i.windowManager.FocusWindow(window); err != nil {
		return nil, fmt.Errorf("failed to focus window: %w", err)
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
		return nil, fmt.Errorf("failed to read clipboard: %w", err)
	}
	if clipboardText == "" {
		return nil, fmt.Errorf("no item text found in clipboard")
	}

	i.log.Debug("Extracted item text", "text", clipboardText)

    // Parse the full item data (reusing existing parsing logic)
    statsmap.Load()
    itemData, err := i.parseItemData(clipboardText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse item data: %w", err)
	}

	i.log.Debug("Parsed item data for price check", "item", itemData)

	// Generate and print the trade search URL for debugging (matches price check query)
	tradeURL := i.buildPriceSearchURL(itemData)
	i.log.Info("Price check search URL for debugging", "url", tradeURL)
	fmt.Printf("\n🔗 Debug: Price check search URL (matches API query)\n%s\n\n", tradeURL)

	// Get price data via API calls
	priceData, err := i.fetchPriceData(itemData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price data: %w", err)
	}

	// Return price data for client display
	result := map[string]interface{}{
		"item_name":           itemData.Name,
		"base_type":           itemData.BaseType,
		"item_class":          itemData.ItemClass,
		"league":              itemData.League,
		"min_price":           priceData.MinPrice,
		"max_price":           priceData.MaxPrice,
		"avg_price":           priceData.AvgPrice,
		"total_listings":      priceData.TotalListings,
		"currency":            priceData.Currency,
		"modifiers_found":     len(itemData.Stats),
	}
	
	// Count searchable modifiers
	matchedStats := 0
	for _, stat := range itemData.Stats {
		if stat.StatID != "" {
			matchedStats++
		}
	}
	result["searchable_modifiers"] = matchedStats
	
	return result, nil
}

// PriceData represents the price information from API
type PriceData struct {
	MinPrice    float64
	MaxPrice    float64
	AvgPrice    float64
	TotalListings int
	Currency    string
}

// TradeAPIResponse represents the structure of PoE trade API response
type TradeAPIResponse struct {
	ID     string `json:"id"`
	Result []struct {
		ID      string `json:"id"`
		Listing struct {
			Method    string `json:"method"`
			Indexed   string `json:"indexed"`
			Stash     struct {
				Name string `json:"name"`
			} `json:"stash"`
			Whisper string `json:"whisper"`
			Account struct {
				Name       string `json:"name"`
				Online     struct {
					League string `json:"league"`
				} `json:"online"`
				LastCharacterName string `json:"lastCharacterName"`
				Language          string `json:"language"`
				Realm             string `json:"realm"`
			} `json:"account"`
			Price struct {
				Type     string  `json:"type"`
				Amount   float64 `json:"amount"`
				Currency string  `json:"currency"`
			} `json:"price"`
		} `json:"listing"`
		Item struct {
			Verified  bool   `json:"verified"`
			W         int    `json:"w"`
			H         int    `json:"h"`
			Icon      string `json:"icon"`
			League    string `json:"league"`
			ID        string `json:"id"`
			Name      string `json:"name"`
			TypeLine  string `json:"typeLine"`
			BaseType  string `json:"baseType"`
			ItemLevel int    `json:"ilvl"`
		} `json:"item"`
	} `json:"result"`
	Total int `json:"total"`
}

// buildPriceQuery creates a trade query specifically optimized for price checking
func (i *Input) buildPriceQuery(item *ItemData) TradeQuery {
	query := TradeQuery{}
	query.Query.Status.Option = "securable"
	query.Sort.Price = "asc"
	query.Query.Stats = make([]StatGroup, 0)

    // Set item category based on Item Class for broader search
    if item.ItemClass != "" {
        category := i.mapItemClassToCategory(item.ItemClass)
        if category != "" {
			if query.Query.Filters.TypeFilters == nil {
				query.Query.Filters.TypeFilters = &struct {
					Filters struct {
						Category *struct {
							Option string `json:"option,omitempty"`
						} `json:"category,omitempty"`
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
            query.Query.Filters.TypeFilters.Filters.Category = &struct {
                Option string `json:"option,omitempty"`
            }{Option: category}
        }
    }

	// Add stat filters (use broader ranges for price checking)
    var currentCategory string
    if item.ItemClass != "" {
        currentCategory = i.mapItemClassToCategory(item.ItemClass)
    }
    statFilters := i.buildPriceStatFilters(item.Stats, currentCategory)
	if len(statFilters) > 0 {
		statGroup := StatGroup{
			Type:    "and",
			Filters: statFilters,
		}
		query.Query.Stats = append(query.Query.Stats, statGroup)
	}
	
	return query
}

// fetchPriceData makes HTTP requests to PoE trade API to get pricing information
func (i *Input) fetchPriceData(item *ItemData) (*PriceData, error) {
	// Build the trade query for price checking (broader ranges)
	query := i.buildPriceQuery(item)

	// Serialize the query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trade query: %w", err)
	}

	// Make the search request to PoE trade API
	baseURL := "https://www.pathofexile.com"
	searchURL := baseURL + "/api/trade2/search/poe2/" + url.PathEscape(item.League)
	
	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(queryJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	// Load POESESSID from environment
	poesessid := os.Getenv("POESESSID")
	if poesessid == "" {
		return nil, fmt.Errorf("POESESSID environment variable is not set")
	}
	req.Header.Set("Cookie", "POESESSID="+poesessid)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform search request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response body: %w", err)
	}
	
	var searchResp struct {
		ID     string   `json:"id"`
		Result []string `json:"result"`
		Total  int      `json:"total"`
	}
	// Debug log the response body to understand what we're getting
	i.log.Debug("Search API response body", "body", string(body), "status", resp.StatusCode)
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}
	
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search response: %w, body: %s", err, string(body))
	}
	
	if searchResp.Total == 0 {
		return &PriceData{
			MinPrice:      0,
			MaxPrice:      0,
			AvgPrice:      0,
			TotalListings: 0,
			Currency:      "chaos",
		}, nil
	}
	
	// Fetch details for first 10 results
	numFetch := 10
	if len(searchResp.Result) < numFetch {
		numFetch = len(searchResp.Result)
	}
	resultIDs := searchResp.Result[:numFetch]
	fetchURL := baseURL + "/api/trade2/fetch/" + strings.Join(resultIDs, ",") + "?query=" + searchResp.ID
	
	fetchReq, err := http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch request: %w", err)
	}
	fetchReq.Header.Set("Cookie", "POESESSID="+poesessid)
	fetchReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	fetchResp, err := client.Do(fetchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to perform fetch request: %w", err)
	}
	defer fetchResp.Body.Close()
	
	fetchBody, err := io.ReadAll(fetchResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read fetch response body: %w", err)
	}
	
	var tradeResp TradeAPIResponse
	// Debug log the fetch response body
	i.log.Debug("Fetch API response body", "body", string(fetchBody), "status", fetchResp.StatusCode)
	
	if fetchResp.StatusCode != 200 {
		return nil, fmt.Errorf("Fetch API returned non-200 status: %d, body: %s", fetchResp.StatusCode, string(fetchBody))
	}
	
	if err := json.Unmarshal(fetchBody, &tradeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fetch response: %w, body: %s", err, string(fetchBody))
	}
	
	var prices []float64
	var currency string
	for _, r := range tradeResp.Result {
		if r.Listing.Price.Amount > 0 {
			prices = append(prices, r.Listing.Price.Amount)
			if currency == "" {
				currency = r.Listing.Price.Currency
			}
		}
	}
	
	if len(prices) == 0 {
		return &PriceData{
			MinPrice:      0,
			MaxPrice:      0,
			AvgPrice:      0,
			TotalListings: searchResp.Total,
			Currency:      "chaos",
		}, nil
	}
	
	sort.Float64s(prices)
	minPrice := prices[0]
	maxPrice := prices[len(prices)-1]
	sum := 0.0
	for _, p := range prices {
		sum += p
	}
	avgPrice := sum / float64(len(prices))
	
	if currency == "" {
		currency = "chaos"
	}
	
	return &PriceData{
		MinPrice:      minPrice,
		MaxPrice:      maxPrice,
		AvgPrice:      avgPrice,
		TotalListings: searchResp.Total,
		Currency:      currency,
	}, nil
}

// buildPriceStatFilters builds stat filters optimized for price checking (broader ranges)
func (i *Input) buildPriceStatFilters(stats []ItemStat, category string) []StatFilter {
    var filters []StatFilter
	
	for _, stat := range stats {
		// Skip stats without stat IDs
		if stat.StatID == "" {
			continue
		}
		
        filter := StatFilter{
            ID:       stat.StatID,
            Disabled: false,
        }

        // Contextual fix for local ES on armour
        if filter.ID == "explicit.stat_3489782002" {
            if strings.HasPrefix(category, "armour.") {
                filter.ID = "explicit.stat_4052037485"
            }
        }
		
		// Use only minimum values for price checking (no maximum constraint)
		if stat.Value > 0 {
			filter.Value = &struct {
				Min *int `json:"min,omitempty"`
				Max *int `json:"max,omitempty"`
			}{}
			
			// Use 95% of the stat value as minimum (find items with at least this much)
			minValue := int(float64(stat.Value) * 0.95)
			
			if minValue < 1 {
				minValue = 1
			}
			
			filter.Value.Min = &minValue
			// Don't set Max - we want items with this stat value OR HIGHER
		}
		
		filters = append(filters, filter)
	}
	
    return filters
}

// displayPriceSummary outputs the price analysis to the console
func (i *Input) displayPriceSummary(item *ItemData, priceData *PriceData) {
	fmt.Printf("\n=== Price Check Results ===\n")
	fmt.Printf("Item: %s\n", item.Name)
	if item.BaseType != "" && item.BaseType != item.Name {
		fmt.Printf("Base Type: %s\n", item.BaseType)
	}
	if item.ItemClass != "" {
		fmt.Printf("Class: %s\n", item.ItemClass)
	}
	fmt.Printf("League: %s\n", item.League)
	fmt.Printf("\n--- Price Analysis ---\n")
	fmt.Printf("Total Listings: %d\n", priceData.TotalListings)
	fmt.Printf("Min Price: %.1f %s\n", priceData.MinPrice, priceData.Currency)
	fmt.Printf("Max Price: %.1f %s\n", priceData.MaxPrice, priceData.Currency)
	fmt.Printf("Average Price: %.1f %s\n", priceData.AvgPrice, priceData.Currency)
	fmt.Printf("\n--- Statistics ---\n")
	fmt.Printf("Modifiers Found: %d\n", len(item.Stats))
	
	// Show stat IDs that were matched for the search
	matchedStats := 0
	for _, stat := range item.Stats {
		if stat.StatID != "" {
			matchedStats++
		}
	}
	fmt.Printf("Searchable Modifiers: %d\n", matchedStats)
	
	if matchedStats == 0 {
		fmt.Printf("\nNote: No modifiers could be matched to trade API IDs.\n")
		fmt.Printf("Price check was based on item category only.\n")
	}
	
	fmt.Printf("========================\n\n")
}

// ItemData represents the parsed item information
type ItemData struct {
	Name        string
	BaseType    string
	ItemClass   string // Item Class: Wands, Armours, etc.
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
		Stats   []StatGroup `json:"stats"`
		Filters struct {
			TypeFilters *struct {
				Filters struct {
					Category *struct {
						Option string `json:"option,omitempty"`
					} `json:"category,omitempty"`
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
        League:       "Rise of the Abyssal", // Default PoE 2 league
    }

	// Remove quantity prefix if present in first line
	firstLine := strings.TrimSpace(lines[0])
	quantityRegex := regexp.MustCompile(`^\d+x\s+`)
	firstLine = quantityRegex.ReplaceAllString(firstLine, "")

	// Parse PoE 2 format: first few lines contain item class, rarity, name, base type
	lineIndex := 0
	
	// Parse "Item Class:" line if present
	if strings.HasPrefix(firstLine, "Item Class:") {
		item.ItemClass = strings.TrimSpace(strings.TrimPrefix(firstLine, "Item Class:"))
		i.log.Debug("Parsed item class", "class", item.ItemClass)
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
        "Armour:",
        "Evasion Rating:",
        "Energy Shield:",
        "Grants Skill:",
        "--------",
    }

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(cleanLine, prefix) {
			return nil
		}
	}

	// Skip rune-based modifiers (contains "(rune)" in the text)
	if strings.Contains(cleanLine, "(rune)") {
		i.log.Debug("Skipping rune modifier", "text", cleanLine)
		return nil
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
        // Parse first number; support integer or decimal
        if val, err := strconv.Atoi(numbers[0]); err == nil {
            stat.Value = val
            stat.Min = val
            stat.Max = val
        } else if f, err := strconv.ParseFloat(numbers[0], 64); err == nil {
            iv := int(math.Round(f))
            stat.Value = iv
            stat.Min = iv
            stat.Max = iv
        }

        // If there are two numbers, treat them as min/max range; support decimals
        if len(numbers) >= 2 {
            if min, err := strconv.Atoi(numbers[0]); err == nil {
                stat.Min = min
            } else if f, err := strconv.ParseFloat(numbers[0], 64); err == nil {
                stat.Min = int(math.Round(f))
            }
            if max, err := strconv.Atoi(numbers[1]); err == nil {
                stat.Max = max
            } else if f, err := strconv.ParseFloat(numbers[1], 64); err == nil {
                stat.Max = int(math.Round(f))
            }
            stat.Value = (stat.Min + stat.Max) / 2 // Use average as main value
            stat.IsRange = true
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

// normalizeToMatcher converts a raw stat text to the Exiled-Exchange matcher format
// by replacing numeric literals with '#' and collapsing whitespace.
func normalizeToMatcher(s string) string {
    // Remove color/format braces if any remain
    s = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(s, "")
    s = strings.TrimSpace(s)
    // Replace numbers (including optional sign or decimal) with '#'
    s = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`).ReplaceAllString(s, "#")
    // Normalize whitespace
    s = strings.Join(strings.Fields(s), " ")
    return s
}

// classifyModifier attempts to classify a modifier and assign a stat ID based on common PoE 2 patterns
func (i *Input) classifyModifier(stat *ItemStat) {
    // First, try to resolve using Exiled-Exchange-2 data if available
    norm := normalizeToMatcher(stat.Text)
    if id, ok := statsmap.FindID(norm); ok {
        stat.StatID = id
        // We don't need exact prefix/suffix for trade, but try a simple guess
        if strings.Contains(strings.ToLower(stat.Text), "resistance") ||
            strings.Contains(strings.ToLower(stat.Text), "accuracy") ||
            strings.Contains(strings.ToLower(stat.Text), "critical") ||
            strings.Contains(strings.ToLower(stat.Text), "speed") ||
            strings.Contains(strings.ToLower(stat.Text), "intelligence") ||
            strings.Contains(strings.ToLower(stat.Text), "strength") ||
            strings.Contains(strings.ToLower(stat.Text), "dexterity") {
            stat.ModifierType = "suffix"
        } else {
            stat.ModifierType = "prefix"
        }
        i.log.Debug("Matched stat via external mapping", "text", stat.Text, "matcher", norm, "id", id)
        return
    }
    // Using limited built-in patterns as fallback for PoE 2 trade API compatibility
	
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
		
		// Fire Damage (Prefix) - explicit.stat_3962278098
		{regexp.MustCompile(`(\d+)% increased Fire Damage`), "explicit.stat_3962278098", "prefix", "Increased Fire Damage"},
		
		// Energy Shield (Prefix) - explicit.stat_4015621042
		{regexp.MustCompile(`(\d+)% increased Energy Shield`), "explicit.stat_4015621042", "prefix", "Increased Energy Shield"},
		
		// Resistances (Suffix) - with specific stat IDs
		{regexp.MustCompile(`\+(\d+)% to Fire Resistance`), "explicit.stat_3372524247", "suffix", "Fire Resistance"},
		{regexp.MustCompile(`\+(\d+)% to Lightning Resistance`), "explicit.stat_1671376347", "suffix", "Lightning Resistance"},
		{regexp.MustCompile(`\+(\d+)% to Cold Resistance`), "explicit.stat_4220027924", "suffix", "Cold Resistance"},
		
		// Life (Prefix) - Need to look up the actual ID
		{regexp.MustCompile(`\+(\d+) to maximum Life`), "", "prefix", "Maximum Life"},
		
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

    // Try to match against built-in patterns (fallback)
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
	
	// Check for prefix patterns (classification only, no stat ID assignment)
	for _, indicator := range prefixIndicators {
		if strings.Contains(text, indicator) {
			stat.ModifierType = "prefix"
			stat.StatID = "" // Don't assign invalid stat IDs
			i.log.Debug("Classified as prefix by convention", "text", stat.Text, "indicator", indicator)
			return
		}
	}
	
	// Check for suffix patterns (classification only, no stat ID assignment)
	for _, indicator := range suffixIndicators {
		if strings.Contains(text, indicator) {
			stat.ModifierType = "suffix" 
			stat.StatID = "" // Don't assign invalid stat IDs
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
func (i *Input) buildStatFilters(stats []ItemStat, category string) []StatFilter {
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

        // Contextual fix: prefer local maximum Energy Shield on armour pieces
        if filter.ID == "explicit.stat_3489782002" { // generic max ES
            if strings.HasPrefix(category, "armour.") {
                filter.ID = "explicit.stat_4052037485" // local max ES (armour)
                i.log.Debug("Adjusted stat to local ES for armour", "from", "explicit.stat_3489782002", "to", filter.ID, "text", stat.Text)
            }
        }
		
		// Add value constraints based on the stat values with ±10% range
		if stat.Value > 0 {
			filter.Value = &struct {
				Min *int `json:"min,omitempty"`
				Max *int `json:"max,omitempty"`
			}{}
			
			// Use ±10% range around the actual stat value for better matching
			minValue := int(float64(stat.Value) * 0.9)
			maxValue := int(float64(stat.Value) * 1.1)
			
			// Ensure minimum is at least 1 for positive stats
			if minValue < 1 {
				minValue = 1
			}
			
			filter.Value.Min = &minValue
			filter.Value.Max = &maxValue
			
			i.log.Debug("Added ranged stat filter", 
				"id", stat.StatID, 
				"min", minValue, 
				"max", maxValue, 
				"original", stat.Value,
				"text", stat.Text)
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

// mapItemClassToCategory maps PoE 2 item classes to API category format
func (i *Input) mapItemClassToCategory(itemClass string) string {
	// Map item classes to API category format based on the API structure
	categoryMap := map[string]string{
		"Wands":           "weapon.wand",
		"Swords":          "weapon.sword",
		"Axes":            "weapon.axe",
		"Maces":           "weapon.mace",
		"Daggers":         "weapon.dagger",
		"Claws":           "weapon.claw",
		"Bows":            "weapon.bow",
		"Crossbows":       "weapon.crossbow",
		"Staves":          "weapon.staff",
		"Sceptres":        "weapon.sceptre",
		"Flails":          "weapon.flail",
		"Spears":          "weapon.spear",
		"Shields":         "armour.shield",
		"Helmets":         "armour.helmet",
		"Body Armours":    "armour.chest",
		"Gloves":          "armour.gloves",
		"Boots":           "armour.boots",
		"Belts":           "accessory.belt",
		"Rings":           "accessory.ring",
		"Amulets":         "accessory.amulet",
		"Jewels":          "jewel",
		"Maps":            "map",
		"Currency":        "currency",
		"Divination Cards": "card",
		"Gems":            "gem",
		"Foci":            "weapon.focus",
	}
	
	if category, exists := categoryMap[itemClass]; exists {
		i.log.Debug("Mapped item class to category", "class", itemClass, "category", category)
		return category
	}
	
	i.log.Debug("No category mapping found for item class", "class", itemClass)
	return ""
}

// buildAdvancedTradeSearchURL constructs a PoE 2 trade site URL with comprehensive search parameters
func (i *Input) buildAdvancedTradeSearchURL(item *ItemData) string {
	query := TradeQuery{}
	
	// Basic query setup
	query.Query.Status.Option = "securable"
	query.Sort.Price = "asc"
	query.Query.Stats = make([]StatGroup, 0)

    // Set item category based on Item Class for broader search
    if item.ItemClass != "" {
        category := i.mapItemClassToCategory(item.ItemClass)
        if category != "" {
			if query.Query.Filters.TypeFilters == nil {
				query.Query.Filters.TypeFilters = &struct {
					Filters struct {
						Category *struct {
							Option string `json:"option,omitempty"`
						} `json:"category,omitempty"`
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
            query.Query.Filters.TypeFilters.Filters.Category = &struct {
                Option string `json:"option,omitempty"`
            }{Option: category}
            i.log.Debug("Set category filter", "category", category)
        }
    }

    // Intentionally ignore quality to broaden price checks

	// Removed item level and corruption filters for broader search results

	// Add stat filters from parsed modifiers
    // Build stat filters with context (category)
    var currentCategory string
    if item.ItemClass != "" {
        currentCategory = i.mapItemClassToCategory(item.ItemClass)
    }
    statFilters := i.buildStatFilters(item.Stats, currentCategory)
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
        return i.buildSimpleTradeSearchURL(item.League, item.Name)
    }

	// Construct the final URL
    baseURL := fmt.Sprintf("https://www.pathofexile.com/trade2/search/poe2/%s", url.PathEscape(item.League))
	encodedQuery := url.QueryEscape(string(queryJSON))
	
	return fmt.Sprintf("%s?q=%s", baseURL, encodedQuery)
}

// buildPriceSearchURL constructs a PoE 2 trade site URL using the same query as price checking
func (i *Input) buildPriceSearchURL(item *ItemData) string {
	// Use the same query structure as price checking for exact debugging match
	query := i.buildPriceQuery(item)

	// Serialize the query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		i.log.Error("Failed to marshal price query for URL", err)
        // Fallback to simple search
        return i.buildSimpleTradeSearchURL(item.League, item.Name)
    }

	// Construct the final URL
    baseURL := fmt.Sprintf("https://www.pathofexile.com/trade2/search/poe2/%s", url.PathEscape(item.League))
	encodedQuery := url.QueryEscape(string(queryJSON))
	
	return fmt.Sprintf("%s?q=%s", baseURL, encodedQuery)
}

// buildSimpleTradeSearchURL constructs a simple PoE 2 trade site URL as fallback
func (i *Input) buildSimpleTradeSearchURL(league string, itemName string) string {
    baseURL := fmt.Sprintf("https://www.pathofexile.com/trade2/search/poe2/%s", url.PathEscape(league))
	
	// Create a simple query structure
	simpleQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"status": map[string]string{"option": "securable"},
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
