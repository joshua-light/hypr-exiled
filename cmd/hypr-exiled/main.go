package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"hypr-exiled/internal/app"
	"hypr-exiled/internal/ipc"
	"hypr-exiled/pkg/config"
	"hypr-exiled/pkg/global"
	"hypr-exiled/pkg/logger"
	"hypr-exiled/pkg/notify"
)

//go:embed assets/*
var embeddedAssets embed.FS

func main() {
	// Load environment variables from .env file if it exists
	_ = godotenv.Load()
	
	configPath := flag.String("config", "", "path to config file")
	debug := flag.Bool("debug", false, "enable debug logging")
	showTrades := flag.Bool("showTrades", false, "show the trades UI")
	hideout := flag.Bool("hideout", false, "go to hideout")
	kingsmarch := flag.Bool("kingsmarch", false, "go to kingsmarch")
    search := flag.Bool("search", false, "search item on PoE 2 trade site")
    price := flag.Bool("price", false, "check average price for item via API")
    research := flag.Bool("research", false, "research high-priced items for the same type and aggregate impactful stats")
	flag.Parse()

	// Initialize logger
	logLevel := zerolog.InfoLevel
	if *debug {
		logLevel = zerolog.DebugLevel
	}

	log, err := logger.NewLogger(
		logger.WithConsole(),
		logger.WithLevel(logLevel),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger failed: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	// Route commands
    switch {
	case *showTrades:
		handleShowTrades(log, *configPath)
	case *hideout:
		handleHideout(log, *configPath)
	case *kingsmarch:
		handleKingsmarch(log, *configPath)
	case *search:
		handleSearch(log, *configPath)
    case *price:
        handlePrice(log, *configPath)
    case *research:
        handleResearch(log, *configPath)
    default:
        startBackgroundService(log, *configPath)
    }
}

// handleShowTrades handles the --showTrades command.
func handleShowTrades(log *logger.Logger, configPath string) {
	log.Info("Showing trades UI")
	_, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer cleanup()

	resp, err := ipc.SendCommand("showTrades")
	if err != nil {
		log.Error("Failed to communicate with background service", err)
		global.GetNotifier().Show("Failed to communicate with background service. Is it running?", notify.Error)
		return
	}

	if resp.Status != "success" {
		log.Error("Failed to show trades", fmt.Errorf("message: %s", resp.Message))
		global.GetNotifier().Show(resp.Message, notify.Error)
		return
	}

	log.Info("Trades displayed successfully")
}

// startBackgroundService starts the background service.
func startBackgroundService(log *logger.Logger, configPath string) {
	cfg, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		os.Exit(1)
	}
	defer cleanup()

	// Create and start service
	log.Info("Service configuration loaded",
		"poe_log_path", cfg.GetPoeLogPath(),
		"triggers", len(cfg.GetTriggers()),
		"commands", len(cfg.GetCommands()))

	app, err := app.NewHyprExiled()
	if err != nil {
		log.Fatal("Failed to create Hypr Exiled", err)
	}

	log.Info("Starting application")
	if err := app.Run(); err != nil {
		log.Fatal("Application error", err)
	}
}

func handleHideout(log *logger.Logger, configPath string) {
	_, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer cleanup()

	resp, err := ipc.SendCommand("hideout")
	if err != nil {
		log.Error("Hideout command failed", err)
		global.GetNotifier().Show("Failed to contact service", notify.Error)
		return
	}

	if resp.Status != "success" {
		log.Error("Hideout failed", fmt.Errorf(resp.Message))
		global.GetNotifier().Show(resp.Message, notify.Error)
		return
	}

	log.Info("Hideout command executed via IPC")
}

func handleKingsmarch(log *logger.Logger, configPath string) {
	_, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer cleanup()

	resp, err := ipc.SendCommand("kingsmarch")
	if err != nil {
		log.Error("Kingsmarch command failed", err)
		global.GetNotifier().Show("Failed to contact service", notify.Error)
		return
	}

	if resp.Status != "success" {
		log.Error("Kingsmarch failed", fmt.Errorf(resp.Message))
		global.GetNotifier().Show(resp.Message, notify.Error)
		return
	}

	log.Info("Kingsmarch command executed via IPC")
}

func handleSearch(log *logger.Logger, configPath string) {
	_, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}
	defer cleanup()

	resp, err := ipc.SendCommand("search")
	if err != nil {
		log.Error("Search command failed", err)
		global.GetNotifier().Show("Failed to contact service", notify.Error)
		return
	}

	if resp.Status != "success" {
		log.Error("Search failed", fmt.Errorf(resp.Message))
		global.GetNotifier().Show(resp.Message, notify.Error)
		return
	}

	log.Info("Search command executed via IPC")
}

func handlePrice(log *logger.Logger, configPath string) {
	log.Info("Starting price check command")
	_, cleanup, err := initializeCommon(log, configPath)
	if err != nil {
		log.Error("Initialization failed", err)
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		global.GetNotifier().Show("Price check failed: "+err.Error(), notify.Error)
		return
	}
	defer cleanup()

	log.Debug("Sending price command to background service")
	resp, err := ipc.SendCommand("price")
	if err != nil {
		log.Error("Price command failed", err)
		global.GetNotifier().Show("Failed to contact service", notify.Error)
		return
	}

	if resp.Status != "success" {
		log.Error("Price check failed", fmt.Errorf(resp.Message))
		global.GetNotifier().Show(resp.Message, notify.Error)
		return
	}

	// Display price data if available
	if resp.PriceData != nil {
		displayPriceResults(resp.PriceData)
		showPriceNotification(resp.PriceData)
	}

	log.Info("Price command executed via IPC")
}

func handleResearch(log *logger.Logger, configPath string) {
    log.Info("Starting research command")
    _, cleanup, err := initializeCommon(log, configPath)
    if err != nil {
        log.Error("Initialization failed", err)
        fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
        global.GetNotifier().Show("Research failed: "+err.Error(), notify.Error)
        return
    }
    defer cleanup()

    log.Debug("Sending research command to background service")
    resp, err := ipc.SendCommand("research")
    if err != nil {
        log.Error("Research command failed", err)
        global.GetNotifier().Show("Failed to contact service", notify.Error)
        return
    }

    if resp.Status != "success" {
        log.Error("Research failed", fmt.Errorf(resp.Message))
        global.GetNotifier().Show(resp.Message, notify.Error)
        return
    }

    // Log a concise summary to the console logger
    if resp.ResearchData != nil {
        league, _ := resp.ResearchData["league"].(string)
        itemClass, _ := resp.ResearchData["item_class"].(string)
        category, _ := resp.ResearchData["category"].(string)
        currency, _ := resp.ResearchData["currency"].(string)
        totalListings, _ := resp.ResearchData["total_listings"].(float64)
        consideredListings, _ := resp.ResearchData["considered_listings"].(float64)
        statsAny, hasStats := resp.ResearchData["stats"].([]interface{})
        statCount := 0
        if hasStats {
            statCount = len(statsAny)
        }

        log.Info("Research completed",
            "league", league,
            "item_class", itemClass,
            "category", category,
            "currency", currency,
            "total_listings", fmt.Sprintf("%.0f", totalListings),
            "considered", fmt.Sprintf("%.0f", consideredListings),
            "stats", statCount,
        )

        // Print detailed table to stdout similar to price
        displayResearchResults(resp.ResearchData)
        showResearchNotification(resp.ResearchData)
    } else {
        log.Info("Research returned no data payload")
    }

    log.Info("Research command executed via IPC")
}

func displayResearchResults(data map[string]interface{}) {
    fmt.Printf("\n=== Research Results ===\n")
    if league, ok := data["league"].(string); ok && league != "" {
        fmt.Printf("League: %s\n", league)
    }
    if itemClass, ok := data["item_class"].(string); ok && itemClass != "" {
        fmt.Printf("Class: %s\n", itemClass)
    }
    if category, ok := data["category"].(string); ok && category != "" {
        fmt.Printf("Category: %s\n", category)
    }
    if currency, ok := data["currency"].(string); ok && currency != "" {
        fmt.Printf("Currency: %s\n", currency)
    }
    if total, ok := data["total_listings"].(float64); ok {
        fmt.Printf("Total Listings (API): %.0f\n", total)
    }
    if considered, ok := data["considered_listings"].(float64); ok {
        fmt.Printf("Considered Listings: %.0f\n", considered)
    }

    fmt.Printf("\n--- All Stats by Weighted Score ---\n")
    if stats, ok := data["stats"].([]interface{}); ok {
        for idx := 0; idx < len(stats); idx++ {
            s, _ := stats[idx].(map[string]interface{})
            id, _ := s["id"].(string)
            text, _ := s["text"].(string)
            count, _ := s["count"].(float64)
            wscore, _ := s["weighted_score"].(float64)
            avg, _ := s["avg"].(float64)
            min, _ := s["min"].(float64)
            max, _ := s["max"].(float64)
            coverage, _ := s["coverage_pct"].(float64)
            fmt.Printf("%2d. %s (%s) | w=%.2f | avg=%.1f min=%.0f max=%.0f | seen=%.0f (%.1f%%)\n",
                idx+1, text, id, wscore, avg, min, max, count, coverage)
        }
    }

    // Print unmatched stats if any to help diagnose missing mappings
    if um, ok := data["unmatched_stats"].([]interface{}); ok && len(um) > 0 {
        fmt.Printf("\n--- Unmatched Mods (not mapped to API IDs) ---\n")
        for idx := 0; idx < len(um); idx++ {
            s, _ := um[idx].(map[string]interface{})
            text, _ := s["text"].(string)
            count, _ := s["count"].(float64)
            wscore, _ := s["weighted_score"].(float64)
            coverage, _ := s["coverage_pct"].(float64)
            fmt.Printf("- %s | w=%.2f | seen=%.0f (%.1f%%)\n", text, wscore, count, coverage)
        }
    }

    fmt.Printf("========================\n\n")
}

func showResearchNotification(data map[string]interface{}) {
    // Build a short summary notification
    league, _ := data["league"].(string)
    itemClass, _ := data["item_class"].(string)
    considered, _ := data["considered_listings"].(float64)
    total, _ := data["total_listings"].(float64)

    top := ""
    if stats, ok := data["stats"].([]interface{}); ok && len(stats) > 0 {
        if s, ok2 := stats[0].(map[string]interface{}); ok2 {
            if t, ok3 := s["text"].(string); ok3 {
                top = t
            }
        }
    }

    msg := fmt.Sprintf("🔬 %s • %s\nItems: %.0f/%.0f • Top: %s", itemClass, league, considered, total, top)
    global.GetNotifier().Show(msg, notify.Info)
}

func displayPriceResults(priceData map[string]interface{}) {
	fmt.Printf("\n=== Price Check Results ===\n")
	
	if itemName, ok := priceData["item_name"].(string); ok && itemName != "" {
		fmt.Printf("Item: %s\n", itemName)
	}
	
	if baseType, ok := priceData["base_type"].(string); ok && baseType != "" {
		if itemName, _ := priceData["item_name"].(string); baseType != itemName {
			fmt.Printf("Base Type: %s\n", baseType)
		}
	}
	
	if itemClass, ok := priceData["item_class"].(string); ok && itemClass != "" {
		fmt.Printf("Class: %s\n", itemClass)
	}
	
	if league, ok := priceData["league"].(string); ok && league != "" {
		fmt.Printf("League: %s\n", league)
	}
	
	fmt.Printf("\n--- Price Analysis ---\n")
	
	if totalListings, ok := priceData["total_listings"].(float64); ok {
		fmt.Printf("Total Listings: %.0f\n", totalListings)
	}
	
	currency := "unknown"
	if c, ok := priceData["currency"].(string); ok {
		currency = c
	}
	
	if minPrice, ok := priceData["min_price"].(float64); ok {
		fmt.Printf("Min Price: %.1f %s\n", minPrice, currency)
	}
	
	if maxPrice, ok := priceData["max_price"].(float64); ok {
		fmt.Printf("Max Price: %.1f %s\n", maxPrice, currency)
	}
	
	if avgPrice, ok := priceData["avg_price"].(float64); ok {
		fmt.Printf("Average Price: %.1f %s\n", avgPrice, currency)
	}
	
	fmt.Printf("\n--- Statistics ---\n")
	
	if modifiersFound, ok := priceData["modifiers_found"].(float64); ok {
		fmt.Printf("Modifiers Found: %.0f\n", modifiersFound)
	}
	
	if searchableModifiers, ok := priceData["searchable_modifiers"].(float64); ok {
		fmt.Printf("Searchable Modifiers: %.0f\n", searchableModifiers)
		
		if searchableModifiers == 0 {
			fmt.Printf("\nNote: No modifiers could be matched to trade API IDs.\n")
			fmt.Printf("Price check was based on item category only.\n")
		}
	}
	
	fmt.Printf("========================\n\n")
}

func showPriceNotification(priceData map[string]interface{}) {
	// Build a concise notification message
	var itemName string
	if name, ok := priceData["item_name"].(string); ok && name != "" {
		itemName = name
	} else {
		itemName = "Item"
	}
	
	var currency string = "unknown"
	if c, ok := priceData["currency"].(string); ok {
		currency = c
	}
	
	var minPrice, maxPrice, avgPrice float64
	if min, ok := priceData["min_price"].(float64); ok {
		minPrice = min
	}
	if max, ok := priceData["max_price"].(float64); ok {
		maxPrice = max
	}
	if avg, ok := priceData["avg_price"].(float64); ok {
		avgPrice = avg
	}
	
	var totalListings float64
	if listings, ok := priceData["total_listings"].(float64); ok {
		totalListings = listings
	}
	
	// Create a concise message for notification
	message := fmt.Sprintf("💰 %s\n%.0f listings: %.1f - %.1f %s\nAvg: %.1f %s", 
		itemName, totalListings, minPrice, maxPrice, currency, avgPrice, currency)
	
	// Show notification
	global.GetNotifier().Show(message, notify.Info)
}

func initializeCommon(log *logger.Logger, configPath string) (*config.Config, func(), error) {
	// Load configuration
	log.Debug("Loading configuration", "path", configPath)
	cfg, err := config.FindConfig(configPath, log, embeddedAssets)
	if err != nil {
		return nil, nil, fmt.Errorf("config error: %w", err)
	}

	// Initialize global state
	log.Debug("Initializing global instances")
	global.InitGlobals(cfg, log, embeddedAssets)

	// Return cleanup function to close resources
	cleanup := func() {
		global.Close()
	}

	return cfg, cleanup, nil
}
