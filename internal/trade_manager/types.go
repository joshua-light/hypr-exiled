package trade_manager

import (
	"sync"

	"hypr-exiled/internal/rofi"
	"hypr-exiled/internal/storage"
	"hypr-exiled/pkg/logger"
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
	db   *storage.DB
	rofi *rofi.TradeDisplayManager
	mu   sync.RWMutex
	log  *logger.Logger
}

type Currency struct {
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Amount int    `json:"amount"`
}
