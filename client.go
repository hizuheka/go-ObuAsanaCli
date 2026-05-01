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

// TaskData はタスク作成時のリクエストペイロードを定義します。
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

// TaskResponseData はタスク一覧取得時のレスポンスペイロードを定義します。
type TaskResponseData struct {
	GID       string `json:"gid"`
	Name      string `json:"name"`
	Completed bool   `json:"completed"`
	DueOn     string `json:"due_on"`
	// Assignee情報はネストされているため、専用の構造体で受け取ります
	AssigneeData *struct {
		GID string `json:"gid"`
	} `json:"assignee"`
}

// Assignee は、ネストされたデータ構造から担当者のGIDを安全に取得するヘルパーメソッドです。
// 純粋関数的なアプローチでNil Panicを防ぎます。
func (t *TaskResponseData) Assignee() string {
	if t.AssigneeData != nil {
		return t.AssigneeData.GID
	}
	return ""
}

// AsanaClient はAsana APIとの通信を抽象化するインターフェースです。
// テスト時にモックと差し替えることで、外部通信なしでロジックのテストが可能になります。
type AsanaClient interface {
	CreateTask(ctx context.Context, task TaskData) (permalinkURL string, err error)
	GetTasks(ctx context.Context, projectGID string) ([]TaskResponseData, error)
}

// asanaHTTPClient は AsanaClient インターフェースの標準的なHTTP実装です。
type asanaHTTPClient struct {
	pat    string
	client *http.Client
}

// NewAsanaClient は新しい AsanaClient を生成します。
func NewAsanaClient(pat string) AsanaClient {
	return &asanaHTTPClient{
		pat: pat,
		// Asanaの複雑なプロジェクトルール等の処理遅延を考慮し、十分な待機時間を設けます
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateTask は指定されたデータで新しいタスクを作成し、パーマリンクのURLを返します。
func (c *asanaHTTPClient) CreateTask(ctx context.Context, task TaskData) (string, error) {
	reqBody := taskRequest{Data: task}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("リクエストの構築に失敗しました: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://app.asana.com/api/1.0/tasks", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("HTTPリクエストの作成に失敗しました: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.pat)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("APIとの通信に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("APIがエラーを返しました (ステータス: %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			PermalinkURL string `json:"permalink_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
	}

	return result.Data.PermalinkURL, nil
}

// GetTasks は指定されたプロジェクトのタスク一覧を取得します。
func (c *asanaHTTPClient) GetTasks(ctx context.Context, projectGID string) ([]TaskResponseData, error) {
	// 必要なフィールドだけをリクエストし、ペイロードサイズと処理時間を最小化します（opt_fieldsの活用）
	url := fmt.Sprintf("https://app.asana.com/api/1.0/tasks?project=%s&opt_fields=name,completed,due_on,assignee", projectGID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストの作成に失敗しました: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.pat)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("APIとの通信に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("APIがエラーを返しました (ステータス: %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []TaskResponseData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("レスポンスの解析に失敗しました: %w", err)
	}

	return result.Data, nil
}
