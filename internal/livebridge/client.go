package livebridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	desc       Descriptor
	httpClient *http.Client
}

func LoadDescriptor() (Descriptor, error) {
	data, err := os.ReadFile(DescriptorPath())
	if err != nil {
		return Descriptor{}, fmt.Errorf("no live tradez bridge found; start the TUI first: %w", err)
	}
	var desc Descriptor
	if err := json.Unmarshal(data, &desc); err != nil {
		return Descriptor{}, fmt.Errorf("invalid bridge descriptor: %w", err)
	}
	if desc.URL == "" || desc.Token == "" || desc.PID == 0 {
		return Descriptor{}, fmt.Errorf("bridge descriptor is incomplete")
	}
	return desc, nil
}

func NewClient(desc Descriptor) *Client {
	return &Client{
		desc: desc,
		httpClient: &http.Client{
			Timeout: 6 * time.Second,
		},
	}
}

func Connect() (*Client, error) {
	desc, err := LoadDescriptor()
	if err != nil {
		return nil, err
	}
	return NewClient(desc), nil
}

func (c *Client) Call(ctx context.Context, op Operation, payload any, out any) error {
	body, err := json.Marshal(struct {
		Operation Operation `json:"operation"`
		Payload   any       `json:"payload,omitempty"`
	}{Operation: op, Payload: payload})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.desc.URL, "/")+"/rpc", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.desc.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("live tradez bridge is unavailable or stale: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("live tradez bridge returned %s", resp.Status)
	}

	var res rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fmt.Errorf("invalid bridge response: %w", err)
	}
	if !res.OK {
		return errors.New(res.Error)
	}
	if out == nil || len(res.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(res.Data, out); err != nil {
		return fmt.Errorf("invalid bridge payload: %w", err)
	}
	return nil
}
