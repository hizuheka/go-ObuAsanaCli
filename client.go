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

// TaskData はAsanaへ送信するタスクモデルです
type TaskData struct {
	Name      string   `json:"name"`
	Workspace string   `json:"workspace"`
	Projects  []string `json:"projects"`
	Notes     string   `json:"notes,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	DueOn     string   `json:"due_on,omitempty"`
}

type taskRequest struct {
	Data TaskData `json:"data"`
}

// AsanaClient はAPI通信を抽象化し、テスト時のモック化を可能にします
type AsanaClient interface {
	CreateTask(ctx context.Context, task TaskData) (permalinkURL string, err error)
}

type asanaHTTPClient struct {
	pat    string
	client *http.Client
}

func NewAsanaClient(pat string) AsanaClient {
	return &asanaHTTPClient{
		pat:    pat,
		client: &http.Client{Timeout: 30 * time.Second}, // タイムアウトを30秒に延長
	}
}

func (c *asanaHTTPClient) CreateTask(ctx context.Context, task TaskData) (string, error) {
	reqBody := taskRequest{Data: task}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://app.asana.com/api/1.0/tasks", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.pat)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			PermalinkURL string `json:"permalink_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Data.PermalinkURL, nil
}
