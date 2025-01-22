package trade_manager

import (
	"net"
	"os/exec"
	"sync"
)

type RofiConfig struct {
	Args    []string
	Message string
}

type Trade struct {
	ID         string     `json:"id"`
	IsSell     bool       `json:"is_sell"`
	Item       string     `json:"item"`
	Price      []Currency `json:"price"`
	PlayerName string     `json:"player_name"`
}

type TradeManager struct {
	trades     []Trade
	mu         sync.RWMutex
	socketPath string
	listener   net.Listener
	rofiCmd    *exec.Cmd
}

type Currency struct {
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Amount int    `json:"amount"`
}
