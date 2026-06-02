package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func httpPostRequest(ctx context.Context, url, body string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

var sharedHTTPClient = &http.Client{Timeout: 10 * time.Second}

func httpDo(req *http.Request) (string, error) {
	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(b), nil
}

func parseJSON(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
