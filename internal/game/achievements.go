package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// AchievementDef describes a single achievement.
type AchievementDef struct {
	ID   string
	Icon string
	Name string
	Desc string
}

// AllAchievements lists every achievement in display order.
var AllAchievements = []AchievementDef{
	{
		ID:   "diamond_hands",
		Icon: "💎",
		Name: "Diamond Hands",
		Desc: "Close a long position at a profit after it was 35%+ underwater at some point",
	},
	{
		ID:   "weather_storm",
		Icon: "⛈",
		Name: "Weather the Storm",
		Desc: "Hold a position through a negative news event targeting your sector for 60+ seconds without selling",
	},
	{
		ID:   "sell_the_dip",
		Icon: "📉",
		Name: "Sell the Dip",
		Desc: "Sell a stock 15%+ below your cost basis, then watch it climb 25%+ above your sell price",
	},
	{
		ID:   "bust",
		Icon: "💀",
		Name: "Bust!!!",
		Desc: "Portfolio value falls below 5% of starting capital",
	},
	{
		ID:   "boom",
		Icon: "💣",
		Name: "BOOM",
		Desc: "Close a single long position for 3× your cost basis (200%+ return on one trade)",
	},
	{
		ID:   "squeezed",
		Icon: "🩳",
		Name: "Squeezed",
		Desc: "A short position racks up unrealized losses ≥75% of the margin you posted",
	},
	{
		ID:   "yolo",
		Icon: "🎲",
		Name: "YOLO",
		Desc: "Commit 85%+ of your total portfolio value to a single buy order",
	},
	{
		ID:   "paper_hands",
		Icon: "🧻",
		Name: "Paper Hands",
		Desc: "Sell a position at a loss, then watch the stock climb 20%+ above your sell price",
	},
	{
		ID:   "to_the_moon",
		Icon: "🚀",
		Name: "To The Moon",
		Desc: "A long position you hold is up 100%+ from your average cost",
	},
	{
		ID:   "tendies",
		Icon: "🍗",
		Name: "Tendies",
		Desc: "Realize profit ≥5× starting capital from closing a single position",
	},
	{
		ID:   "guh",
		Icon: "😱",
		Name: "GUH",
		Desc: "A single sell or cover realizes a loss ≥30% of your total portfolio value",
	},
	{
		ID:   "bag_holder",
		Icon: "🛍",
		Name: "Bag Holder",
		Desc: "Hold a position 40%+ in the red for 90 consecutive seconds",
	},
	{
		ID:   "loss_porn",
		Icon: "📸",
		Name: "Loss Porn",
		Desc: "Portfolio hits 70%+ below starting capital — and you're still playing",
	},
	{
		ID:   "wagmi",
		Icon: "🌙",
		Name: "WAGMI",
		Desc: "Portfolio reaches 5× starting capital. We're all gonna make it",
	},
	{
		ID:   "ape_strong",
		Icon: "🦍",
		Name: "Ape Strong",
		Desc: "Hold 8+ simultaneous long positions across different stocks",
	},
	{
		ID:   "i_like_the_stock",
		Icon: "🐒",
		Name: "I Like the Stock",
		Desc: "Hold a position untouched for 5+ minutes — long-term investing by WSB standards",
	},
	{
		ID:   "widow_maker",
		Icon: "☠",
		Name: "Widow Maker",
		Desc: "A single buy order consumes 95%+ of your available cash",
	},
	{
		ID:   "not_financial_advice",
		Icon: "📊",
		Name: "Not Financial Advice",
		Desc: "Make a trade that's the exact opposite of the Analyst's STRONG recommendation",
	},
	{
		ID:   "short_bus",
		Icon: "🚌",
		Name: "Short Bus",
		Desc: "Profit from 3 separate short positions in one session",
	},
	{
		ID:   "infinite_money_glitch",
		Icon: "🖨",
		Name: "Infinite Money Glitch",
		Desc: "Profit on both a long AND a short position on the same ticker in one session",
	},
	{
		ID:   "free_real_estate",
		Icon: "🏠",
		Name: "Free Real Estate",
		Desc: "A limit buy fills 10%+ below your limit price",
	},
}

// AchievementByID returns the def for the given ID, or nil.
func AchievementByID(id string) *AchievementDef {
	for i := range AllAchievements {
		if AllAchievements[i].ID == id {
			return &AllAchievements[i]
		}
	}
	return nil
}

// AchievementStore holds the persistent unlock state.
type AchievementStore struct {
	Unlocked map[string]time.Time `json:"unlocked"`
	path     string
}

func achievementPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tradez", "achievements.json"), nil
}

// LoadAchievements reads the persistent achievement file; returns an empty store on error.
func LoadAchievements() *AchievementStore {
	store := &AchievementStore{Unlocked: make(map[string]time.Time)}
	path, err := achievementPath()
	if err != nil {
		return store
	}
	store.path = path
	data, err := os.ReadFile(path)
	if err != nil {
		return store
	}
	_ = json.Unmarshal(data, &store.Unlocked)
	return store
}

// Save writes the store to disk.
func (s *AchievementStore) Save() {
	if s.path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	data, err := json.MarshalIndent(s.Unlocked, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0644)
}

// Unlock marks an achievement as unlocked. Returns true if it was newly unlocked.
func (s *AchievementStore) Unlock(id string) bool {
	if _, ok := s.Unlocked[id]; ok {
		return false
	}
	s.Unlocked[id] = time.Now()
	s.Save()
	return true
}

// IsUnlocked reports whether the achievement has been unlocked.
func (s *AchievementStore) IsUnlocked(id string) bool {
	_, ok := s.Unlocked[id]
	return ok
}

// UnlockedCount returns the number of unlocked achievements.
func (s *AchievementStore) UnlockedCount() int {
	return len(s.Unlocked)
}
