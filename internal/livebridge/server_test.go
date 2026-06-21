package livebridge

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeSender struct {
	t *testing.T
}

func (f fakeSender) Send(msg tea.Msg) {
	req, ok := msg.(RequestMsg)
	if !ok {
		f.t.Fatalf("unexpected message type %T", msg)
	}
	if req.Operation != OpGameState {
		f.t.Fatalf("operation = %s, want %s", req.Operation, OpGameState)
	}
	req.Reply <- Response{OK: true, Data: map[string]any{"ok": true}}
}

func TestServerWritesDescriptorAndClientCallsRPC(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	srv.SetSender(fakeSender{t: t})
	srv.SetActiveSlot(3)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer srv.Close()

	info, err := os.Stat(DescriptorPath())
	if err != nil {
		t.Fatalf("descriptor missing: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("descriptor permissions = %o, want 600", got)
	}

	desc, err := LoadDescriptor()
	if err != nil {
		t.Fatalf("LoadDescriptor failed: %v", err)
	}
	if desc.ActiveSlot != 3 {
		t.Fatalf("active slot = %d, want 3", desc.ActiveSlot)
	}

	var out map[string]bool
	if err := NewClient(desc).Call(context.Background(), OpGameState, nil, &out); err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if !out["ok"] {
		raw, _ := json.Marshal(out)
		t.Fatalf("unexpected output: %s", raw)
	}
}
