package livebridge

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Sender interface {
	Send(tea.Msg)
}

type Server struct {
	listener   net.Listener
	httpServer *http.Server
	token      string
	url        string
	pid        int
	sender     atomic.Value
	activeSlot atomic.Int64
}

func NewServer() (*Server, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{
		listener: ln,
		token:    token,
		url:      "http://" + ln.Addr().String(),
		pid:      os.Getpid(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", s.handleRPC)
	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}
	return s, nil
}

func (s *Server) SetSender(sender Sender) {
	s.sender.Store(sender)
}

func (s *Server) Start() error {
	if err := s.writeDescriptor(); err != nil {
		return err
	}
	go func() {
		err := s.httpServer.Serve(s.listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "tradez bridge failed: %v\n", err)
		}
	}()
	return nil
}

func (s *Server) Close() error {
	_ = s.removeDescriptorIfOwned()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) SetActiveSlot(slot int) {
	s.activeSlot.Store(int64(slot))
	_ = s.writeDescriptor()
}

func (s *Server) Descriptor() Descriptor {
	return Descriptor{
		URL:        s.url,
		Token:      s.token,
		PID:        s.pid,
		ActiveSlot: int(s.activeSlot.Load()),
	}
}

func DescriptorPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".tradez", "bridge", "current.json")
	}
	return filepath.Join(home, ".tradez", "bridge", "current.json")
}

func (s *Server) writeDescriptor() error {
	path := DescriptorPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.Descriptor(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (s *Server) removeDescriptorIfOwned() error {
	path := DescriptorPath()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var current Descriptor
	if err := json.Unmarshal(data, &current); err != nil {
		return err
	}
	own := s.Descriptor()
	if current.URL != own.URL || current.Token != own.Token || current.PID != own.PID {
		return nil
	}
	if err := os.Remove(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else {
		return err
	}
}

type rpcRequest struct {
	Operation Operation       `json:"operation"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type rpcResponse struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+s.token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPC(w, rpcResponse{OK: false, Error: "invalid request JSON"})
		return
	}
	if req.Operation == "" {
		writeRPC(w, rpcResponse{OK: false, Error: "missing operation"})
		return
	}

	rawSender := s.sender.Load()
	if rawSender == nil {
		writeRPC(w, rpcResponse{OK: false, Error: "TUI bridge is not ready"})
		return
	}
	sender, ok := rawSender.(Sender)
	if !ok {
		writeRPC(w, rpcResponse{OK: false, Error: "TUI bridge sender is invalid"})
		return
	}

	reply := make(chan Response, 1)
	sender.Send(RequestMsg{
		Operation: req.Operation,
		Payload:   req.Payload,
		Reply:     reply,
	})

	select {
	case res := <-reply:
		if !res.OK {
			writeRPC(w, rpcResponse{OK: false, Error: res.Error})
			return
		}
		data, err := json.Marshal(res.Data)
		if err != nil {
			writeRPC(w, rpcResponse{OK: false, Error: "failed to encode bridge response"})
			return
		}
		writeRPC(w, rpcResponse{OK: true, Data: data})
	case <-time.After(5 * time.Second):
		writeRPC(w, rpcResponse{OK: false, Error: "TUI bridge timed out"})
	}
}

func writeRPC(w http.ResponseWriter, res rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func randomToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
