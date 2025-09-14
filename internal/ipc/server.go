package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"hypr-exiled/internal/input"
	"hypr-exiled/internal/trade_manager"
	"hypr-exiled/pkg/global"
)

const socketPath = "/tmp/hypr-exiled.sock"

type Request struct {
	Command string `json:"command"`
}

type Response struct {
    Status    string                 `json:"status"`
    Message   string                 `json:"message"`
    PriceData map[string]interface{} `json:"price_data,omitempty"`
    ResearchData map[string]interface{} `json:"research_data,omitempty"`
}

func StartSocketServer(tradeManager *trade_manager.TradeManager, input *input.Input) {
	log := global.GetLogger()

	// Remove the socket file if it already exists
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		log.Error("Failed to remove existing socket file", err)
		return
	}

	// Create the directory for the socket file
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal("Failed to create socket directory", err)
	}

	// Listen on the Unix domain socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("Failed to start socket server", err)
	}
	defer listener.Close()

	log.Info("Socket server started", "path", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("Failed to accept connection", err)
			continue
		}

		log.Debug("New connection accepted", "remote_addr", conn.RemoteAddr())

		go handleConnection(conn, tradeManager, input)
	}
}

func handleConnection(conn net.Conn, tradeManager *trade_manager.TradeManager, input *input.Input) {
	log := global.GetLogger()
	defer conn.Close()

	var req Request
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&req); err != nil {
		log.Error("Failed to decode request", err)
		return
	}

	log.Info("Received request", "command", req.Command)

	var resp Response
	switch req.Command {
	case "showTrades":
		log.Debug("Handling showTrades request")
		if err := tradeManager.ShowTrades(); err != nil {
			log.Error("Failed to show trades", err)
			resp = Response{Status: "error", Message: err.Error()}
		} else {
			log.Info("Trades displayed successfully")
			resp = Response{Status: "success", Message: "Trades displayed successfully"}
		}
	case "hideout":
		log.Debug("Handling hideout request")
		if err := input.ExecuteHideout(); err != nil {
			log.Error("Hideout command failed", err)

			resp = Response{
				Status:  "error",
				Message: err.Error(),
			}
		} else {
			log.Info("Hideout command executed successfully")
			resp = Response{
				Status:  "success",
				Message: "Warped to hideout",
			}
		}
	case "kingsmarch":
		log.Debug("Handling kingsmarch request")
		if err := input.ExecuteKingsmarch(); err != nil {
			log.Error("Kingsmarch command failed", err)

			resp = Response{
				Status:  "error",
				Message: err.Error(),
			}
		} else {
			log.Info("Kingsmarch command executed successfully")
			resp = Response{
				Status:  "success",
				Message: "Warped to kingsmarch",
			}
		}
	case "search":
		log.Debug("Handling search request")
		if err := input.ExecuteSearch(); err != nil {
			log.Error("Search command failed", err)

			resp = Response{
				Status:  "error",
				Message: err.Error(),
			}
		} else {
			log.Info("Search command executed successfully")
			resp = Response{
				Status:  "success",
				Message: "Item search opened",
			}
		}
	case "price":
		log.Debug("Handling price request")
		if priceData, err := input.ExecutePrice(); err != nil {
			log.Error("Price command failed", err)

			resp = Response{
				Status:  "error",
				Message: err.Error(),
			}
		} else {
			log.Info("Price command executed successfully")
			resp = Response{
				Status:    "success",
				Message:   "Price check completed",
				PriceData: priceData,
			}
		}
	case "research":
		log.Debug("Handling research request")
		if researchData, err := input.ExecuteResearch(); err != nil {
			log.Error("Research command failed", err)

			resp = Response{
				Status:  "error",
				Message: err.Error(),
			}
		} else {
			// Log a concise summary in the server process as well
			league, _ := researchData["league"].(string)
			itemClass, _ := researchData["item_class"].(string)
			category, _ := researchData["category"].(string)
			total, _ := researchData["total_listings"].(int)
			consideredFloat, _ := researchData["considered_listings"].(int)
			if consideredFloat == 0 {
				if v, ok := researchData["considered_listings"].(float64); ok { consideredFloat = int(v) }
			}
			log.Info("Research command executed successfully",
				"league", league,
				"item_class", itemClass,
				"category", category,
				"total", total,
				"considered", consideredFloat,
			)
			// Also log the top stats (up to 10) if present
			if raw, ok := researchData["stats"].([]map[string]interface{}); ok {
				limit := 10
				for idx := 0; idx < len(raw) && idx < limit; idx++ {
					s := raw[idx]
					log.Info("Top research stat",
						"rank", idx+1,
						"id", s["id"],
						"text", s["text"],
						"weighted_score", s["weighted_score"],
						"coverage_pct", s["coverage_pct"],
					)
				}
			} else if arr, ok := researchData["stats"].([]interface{}); ok {
				limit := 10
				for idx := 0; idx < len(arr) && idx < limit; idx++ {
					if s, ok := arr[idx].(map[string]interface{}); ok {
						log.Info("Top research stat",
							"rank", idx+1,
							"id", s["id"],
							"text", s["text"],
							"weighted_score", s["weighted_score"],
							"coverage_pct", s["coverage_pct"],
						)
					}
				}
			}
			resp = Response{
				Status:       "success",
				Message:      "Research completed",
				ResearchData: researchData,
			}
		}
	default:
		log.Error("Unknown command received", fmt.Errorf("command: %s", req.Command))
		resp = Response{Status: "error", Message: "Unknown command"}
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(resp); err != nil {
		log.Error("Failed to encode response", err)
	} else {
		log.Debug("Response sent successfully", "status", resp.Status)
	}
}
