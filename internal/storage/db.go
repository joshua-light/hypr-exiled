package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hypr-exiled/internal/models"
	"hypr-exiled/pkg/global"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS trades (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    trigger_type TEXT NOT NULL,
    player_name TEXT NOT NULL,
    item_name TEXT NOT NULL,
    league TEXT NOT NULL,
    currency_amount REAL NOT NULL,
    currency_type TEXT NOT NULL,
    stash_tab TEXT NOT NULL,
    position_left INTEGER NOT NULL,
    position_top INTEGER NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

func New() (*DB, error) {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config directory: %w", err)
	}

	// Ensure directory exists
	dbDir := filepath.Join(configDir, "hypr-exiled")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "trades.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) AddTrade(trade models.TradeEntry) error {
	query := `
		INSERT INTO trades (
			timestamp, trigger_type, player_name, item_name, league,
			currency_amount, currency_type, stash_tab,
			position_left, position_top, message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		trade.Timestamp, trade.TriggerType, trade.PlayerName,
		trade.ItemName, trade.League, trade.CurrencyAmount,
		trade.CurrencyType, trade.StashTab, trade.Position.Left,
		trade.Position.Top, trade.Message)

	if err != nil {
		return fmt.Errorf("failed to insert trade: %w", err)
	}

	return nil
}

func (d *DB) GetTrades() ([]models.TradeEntry, error) {
	log := global.GetLogger()
	log.Debug("Retrieving trades from database")

	query := `
        SELECT timestamp, trigger_type, player_name, item_name, league,
               currency_amount, currency_type, stash_tab,
               position_left, position_top, message,
               created_at
        FROM trades
        ORDER BY timestamp DESC
    `

	rows, err := d.db.Query(query)
	if err != nil {
		log.Error("Failed to query trades", err)
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []models.TradeEntry
	for rows.Next() {
		var trade models.TradeEntry
		var timestamp, createdAt time.Time
		err := rows.Scan(
			&timestamp, &trade.TriggerType, &trade.PlayerName,
			&trade.ItemName, &trade.League, &trade.CurrencyAmount,
			&trade.CurrencyType, &trade.StashTab, &trade.Position.Left,
			&trade.Position.Top, &trade.Message, &createdAt)
		if err != nil {
			log.Error("Failed to scan trade", err)
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		trade.Timestamp = timestamp

		log.Debug("Retrieved trade",
			"player_name", trade.PlayerName,
			"item_name", trade.ItemName,
			"trigger_type", trade.TriggerType,
			"timestamp", timestamp,
			"created_at", createdAt)

		trades = append(trades, trade)
	}

	log.Debug("Total trades retrieved", "count", len(trades))
	return trades, nil
}

func (d *DB) RemoveTrades(trades []models.TradeEntry) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, trade := range trades {
		_, err := tx.Exec(`
			DELETE FROM trades 
			WHERE player_name = ? 
			AND item_name = ? 
			AND position_left = ? 
			AND position_top = ?`,
			trade.PlayerName, trade.ItemName,
			trade.Position.Left, trade.Position.Top)
		if err != nil {
			return fmt.Errorf("failed to delete trade: %w", err)
		}
	}

	return tx.Commit()
}

func (d *DB) RemoveTradesByPlayer(playerName string) error {
	_, err := d.db.Exec("DELETE FROM trades WHERE player_name = ?", playerName)
	if err != nil {
		return fmt.Errorf("failed to delete trades for player %s: %w", playerName, err)
	}
	return nil
}

func (d *DB) Cleanup(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := d.db.Exec("DELETE FROM trades WHERE created_at < ?", cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old trades: %w", err)
	}
	return nil
}
