package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"tradez/internal/market"
)

const saveVersion = 1

// SaveSlot holds metadata for a save slot, used to populate the slot-select screen.
type SaveSlot struct {
	Number         int
	Name           string
	Exists         bool
	SavedAt        time.Time
	Cash           float64
	StartingCash   float64
	PortfolioValue float64
	Difficulty     Difficulty
}

type gameState struct {
	Version        int                  `json:"version"`
	SavedAt        time.Time            `json:"saved_at"`
	Cash           float64              `json:"cash"`
	StartingCash   float64              `json:"starting_cash"`
	PortfolioValue float64              `json:"portfolio_value"`
	Difficulty     Difficulty           `json:"difficulty"`
	Positions      map[string]*Position `json:"positions"`
	Orders         []*Order             `json:"orders"`
	NextOrderID    int                  `json:"next_order_id"`
	Messages       []string             `json:"messages"`
	Market         market.MarketState   `json:"market"`
}

func savePath(slot int) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".tradez", "saves")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("slot%d.json", slot)), nil
}

// SlotInfo reads slot metadata without loading the full market state.
func SlotInfo(slot int) SaveSlot {
	info := SaveSlot{Number: slot, Name: fmt.Sprintf("Game %d", slot)}
	path, err := savePath(slot)
	if err != nil {
		return info
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return info
	}
	var gs gameState
	if json.Unmarshal(data, &gs) != nil {
		return info
	}
	info.Exists = true
	info.SavedAt = gs.SavedAt
	info.Cash = gs.Cash
	info.StartingCash = gs.StartingCash
	info.PortfolioValue = gs.PortfolioValue
	info.Difficulty = gs.Difficulty
	return info
}

// Save serializes the complete game state to the given slot (1–3).
func Save(g *Game, slot int) error {
	path, err := savePath(slot)
	if err != nil {
		return err
	}
	gs := gameState{
		Version:        saveVersion,
		SavedAt:        time.Now(),
		Cash:           g.Cash,
		StartingCash:   g.StartingCash,
		PortfolioValue: g.PortfolioValue(),
		Difficulty:     g.Difficulty,
		Positions:      g.Positions,
		Orders:         g.Orders,
		NextOrderID:    g.nextOrderID,
		Messages:       g.Messages,
		Market:         g.Market.Export(),
	}
	data, err := json.Marshal(gs)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load deserializes a game from the given slot (1–3).
func Load(slot int) (*Game, error) {
	path, err := savePath(slot)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var gs gameState
	if err := json.Unmarshal(data, &gs); err != nil {
		return nil, err
	}
	m := market.NewFromState(gs.Market)
	g := &Game{
		Market:       m,
		Cash:         gs.Cash,
		StartingCash: gs.StartingCash,
		Difficulty:   gs.Difficulty,
		Positions:    gs.Positions,
		Orders:       gs.Orders,
		nextOrderID:  gs.NextOrderID,
		Messages:     gs.Messages,
	}
	if g.Positions == nil {
		g.Positions = make(map[string]*Position)
	}
	return g, nil
}
