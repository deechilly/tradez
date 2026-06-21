package livebridge

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
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

	if _, err := Connect(); err != nil {
		t.Fatalf("Connect failed against live bridge: %v", err)
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

func TestConnectRejectsStaleDescriptor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("Close listener failed: %v", err)
	}

	writeTestDescriptor(t, Descriptor{
		URL:        "http://" + addr,
		Token:      "stale-token",
		PID:        os.Getpid(),
		ActiveSlot: 1,
	})

	if _, err := Connect(); err == nil {
		t.Fatal("Connect succeeded for stale descriptor, want error")
	}
}

func TestCloseOnlyRemovesOwnedDescriptor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	first, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer first failed: %v", err)
	}
	first.SetSender(fakeSender{t: t})
	first.SetActiveSlot(1)
	if err := first.Start(); err != nil {
		t.Fatalf("Start first failed: %v", err)
	}

	second, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer second failed: %v", err)
	}
	second.SetSender(fakeSender{t: t})
	second.SetActiveSlot(2)
	if err := second.Start(); err != nil {
		t.Fatalf("Start second failed: %v", err)
	}

	secondDesc := second.Descriptor()
	if err := first.Close(); err != nil {
		t.Fatalf("Close first failed: %v", err)
	}

	current, err := LoadDescriptor()
	if err != nil {
		t.Fatalf("descriptor should still exist after first close: %v", err)
	}
	if current.URL != secondDesc.URL || current.Token != secondDesc.Token || current.PID != secondDesc.PID {
		t.Fatalf("descriptor owner changed after first close: got %#v want %#v", current, secondDesc)
	}

	if err := second.Close(); err != nil {
		t.Fatalf("Close second failed: %v", err)
	}
	if _, err := os.Stat(DescriptorPath()); !os.IsNotExist(err) {
		t.Fatalf("descriptor should be removed by owner close, stat err = %v", err)
	}
}

func writeTestDescriptor(t *testing.T, desc Descriptor) {
	t.Helper()
	path := DescriptorPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	data, err := json.MarshalIndent(desc, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
