package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"hypr-exiled/internal/trade_manager"
	"hypr-exiled/pkg/global"
)

const socketPath = "/tmp/hypr-exiled.sock"

type Request struct {
	Command string `json:"command"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func StartSocketServer(tradeManager *trade_manager.TradeManager) {
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

		go handleConnection(conn, tradeManager)
	}
}

func handleConnection(conn net.Conn, tradeManager *trade_manager.TradeManager) {
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
