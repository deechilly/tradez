package tui

import (
	"testing"
	"tradez/internal/game"
	"tradez/internal/livebridge"
	"tradez/internal/market"
)

func TestBridgeSaveRejectsGameBeforeStartingCapital(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	m := &Model{
		g:          game.New(market.New(1)),
		activeSlot: 1,
	}

	res := m.bridgeResponse(livebridge.RequestMsg{Operation: livebridge.OpSave})
	if res.OK {
		t.Fatal("bridge save succeeded before starting capital was selected")
	}
	if game.SlotInfo(1).Exists {
		t.Fatal("bridge save wrote a save file before starting capital was selected")
	}
}

func TestBridgeSaveAllowsInitializedGame(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	m := &Model{
		g:          game.New(market.New(1)),
		activeSlot: 1,
	}
	m.g.Cash = 50_000
	m.g.StartingCash = 50_000

	res := m.bridgeResponse(livebridge.RequestMsg{Operation: livebridge.OpSave})
	if !res.OK {
		t.Fatalf("bridge save failed for initialized game: %s", res.Error)
	}
	if !game.SlotInfo(1).Exists {
		t.Fatal("bridge save did not write initialized save file")
	}
}
