package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// --- モックの定義 ---

type mockUI struct {
	messages []string
}

func (m *mockUI) Prompt(msg string, req bool) string { return "" }
func (m *mockUI) PromptPassword(msg string) string   { return "" }
func (m *mockUI) Show(msg string)                    { m.messages = append(m.messages, msg) }
func (m *mockUI) Confirm(msg string) bool            { return true }

type mockConfigStore struct {
	cfg *Config
}

func (m *mockConfigStore) Exists() bool           { return true }
func (m *mockConfigStore) Load() (*Config, error) { return m.cfg, nil }
func (m *mockConfigStore) CreateTemplate() error  { return nil }

type mockTokenStore struct{}

func (m *mockTokenStore) Get() (string, error)   { return "test-pat", nil }
func (m *mockTokenStore) Set(token string) error { return nil }
func (m *mockTokenStore) Delete() error          { return nil }

type mockAsanaClient struct {
	tasks []TaskResponseData
	err   error
}

func (m *mockAsanaClient) CreateTask(ctx context.Context, task TaskData) (string, error) {
	return "", nil
}
func (m *mockAsanaClient) GetTasks(ctx context.Context, projectGID string) ([]TaskResponseData, error) {
	return m.tasks, m.err
}

// テスト用の担当者データ生成ヘルパー
func newAssigneeData(gid string) *struct {
	GID string `json:"gid"`
} {
	return &struct {
		GID string `json:"gid"`
	}{GID: gid}
}

// CLIコンテナと出力をトラップするバッファのセットアップ
func setupTestCLI(tasks []TaskResponseData) (*CLI, *bytes.Buffer) {
	cfg := &Config{
		WorkspaceID:    "ws-1",
		Projects:       map[string]string{"dev": "proj-1", "mktg": "proj-2"},
		DefaultProject: "dev",
		Assignees:      map[string]string{"me": "user-1", "john": "user-2"},
	}

	outBuf := new(bytes.Buffer)

	cli := &CLI{
		UI:         &mockUI{},
		Config:     &mockConfigStore{cfg: cfg},
		TokenStore: &mockTokenStore{},
		Client:     &mockAsanaClient{tasks: tasks}, // モックAPIクライアントを注入
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		NowFn:      func() time.Time { return time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC) },
	}

	return cli, outBuf
}

// --- テストケース ---

func TestListCmd_DefaultOutput(t *testing.T) {
	tasks := []TaskResponseData{
		{GID: "1", Name: "Task 1", Completed: false, DueOn: "2026-04-25", AssigneeData: newAssigneeData("user-1")},
		{GID: "2", Name: "Task 2", Completed: true, DueOn: "", AssigneeData: newAssigneeData("user-2")},
		{GID: "3", Name: "Task 3", Completed: false, DueOn: "", AssigneeData: nil},
	}

	cli, outBuf := setupTestCLI(tasks)
	cmd := NewListCmd(cli)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)

	// 引数なし (デフォルト: 未完了のみ)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := outBuf.String()

	if !strings.Contains(output, "Task 1") {
		t.Errorf("expected output to contain 'Task 1'")
	}
	if strings.Contains(output, "Task 2") {
		t.Errorf("expected output NOT to contain 'Task 2' (it is completed)")
	}
	if !strings.Contains(output, "Task 3") {
		t.Errorf("expected output to contain 'Task 3'")
	}
}

func TestListCmd_FilterStatusClosed(t *testing.T) {
	tasks := []TaskResponseData{
		{GID: "1", Name: "Task 1", Completed: false},
		{GID: "2", Name: "Task 2", Completed: true},
	}

	cli, outBuf := setupTestCLI(tasks)
	cmd := NewListCmd(cli)
	cmd.SetOut(outBuf)

	// 引数でステータスを閉じたものに限定
	cmd.SetArgs([]string{"--status", "closed"})
	_ = cmd.Execute()

	output := outBuf.String()
	if strings.Contains(output, "Task 1") {
		t.Errorf("expected output NOT to contain 'Task 1'")
	}
	if !strings.Contains(output, "Task 2") {
		t.Errorf("expected output to contain 'Task 2'")
	}
}

func TestListCmd_FilterAssignee(t *testing.T) {
	tasks := []TaskResponseData{
		{GID: "1", Name: "Task 1", Completed: false, AssigneeData: newAssigneeData("user-1")}, // me に相当
		{GID: "2", Name: "Task 2", Completed: false, AssigneeData: newAssigneeData("user-2")}, // john に相当
	}

	cli, outBuf := setupTestCLI(tasks)
	cmd := NewListCmd(cli)
	cmd.SetOut(outBuf)

	// 設定ファイルで 'me' = 'user-1' にマッピングされている
	cmd.SetArgs([]string{"--assignee", "me"})
	_ = cmd.Execute()

	output := outBuf.String()
	if !strings.Contains(output, "Task 1") {
		t.Errorf("expected output to contain 'Task 1'")
	}
	if strings.Contains(output, "Task 2") {
		t.Errorf("expected output NOT to contain 'Task 2'")
	}
}

func TestListCmd_OutputJSON(t *testing.T) {
	tasks := []TaskResponseData{
		{GID: "1", Name: "JSON Task", Completed: false},
	}

	cli, outBuf := setupTestCLI(tasks)
	cmd := NewListCmd(cli)
	cmd.SetOut(outBuf)

	// JSON出力モードのテスト
	cmd.SetArgs([]string{"--output", "json"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 出力が正しいJSONフォーマットか検証
	var parsed []TaskResponseData
	if err := json.Unmarshal(outBuf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(parsed) != 1 || parsed[0].Name != "JSON Task" {
		t.Errorf("unexpected JSON output: %v", parsed)
	}
}
