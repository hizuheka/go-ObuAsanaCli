package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// --- cmd_add専用のモック定義 ---
// cmd_list_test.go との競合を避けるため、プレフィックスをつけています。

type addMockUI struct {
	inputs      []string
	inputIdx    int
	passwords   []string
	passwordIdx int
	confirms    []bool
	confIdx     int
	messages    []string
}

func (m *addMockUI) Prompt(msg string, req bool) string {
	if m.inputIdx >= len(m.inputs) {
		return ""
	}
	res := m.inputs[m.inputIdx]
	m.inputIdx++
	return res
}
func (m *addMockUI) PromptPassword(msg string) string {
	if m.passwordIdx >= len(m.passwords) {
		return ""
	}
	res := m.passwords[m.passwordIdx]
	m.passwordIdx++
	return res
}
func (m *addMockUI) Show(msg string) {
	m.messages = append(m.messages, msg)
}
func (m *addMockUI) Confirm(msg string) bool {
	if m.confIdx >= len(m.confirms) {
		return false
	}
	res := m.confirms[m.confIdx]
	m.confIdx++
	return res
}

type addMockConfigStore struct {
	cfg    *Config
	exists bool
}

func (m *addMockConfigStore) Exists() bool           { return m.exists }
func (m *addMockConfigStore) Load() (*Config, error) { return m.cfg, nil }
func (m *addMockConfigStore) CreateTemplate() error  { return nil }

type addMockTokenStore struct {
	token string
	err   error
}

func (m *addMockTokenStore) Get() (string, error)   { return m.token, m.err }
func (m *addMockTokenStore) Set(token string) error { m.token = token; return nil }
func (m *addMockTokenStore) Delete() error          { m.token = ""; return nil }

type addMockAsanaClient struct{}

func (m *addMockAsanaClient) CreateTask(ctx context.Context, task TaskData) (string, error) {
	return "https://asana.com/task/1", nil
}
func (m *addMockAsanaClient) GetTasks(ctx context.Context, projectGID string) ([]TaskResponseData, error) {
	return nil, nil
}

// --- テストケース ---

func TestAddCmd_SuccessFlow(t *testing.T) {
	cfg := &Config{
		WorkspaceID:    "ws-1",
		Projects:       map[string]string{"dev": "111"},
		DefaultProject: "dev",
		Assignees:      map[string]string{"me": "123"},
	}

	cli := &CLI{
		UI: &addMockUI{
			inputs: []string{"テストタスク", "", "me", "説明", "today"},
		},
		Config:     &addMockConfigStore{exists: true, cfg: cfg},
		TokenStore: &addMockTokenStore{token: "existing-pat"},
		Client:     &addMockAsanaClient{},
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		NowFn:      func() time.Time { return time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC) },
	}

	cmd := NewAddCmd(cli)
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// UIに渡されたメッセージの中に「登録完了」が含まれているか検証
	mockUI := cli.UI.(*addMockUI)
	output := strings.Join(mockUI.messages, "\n")
	if !strings.Contains(output, "登録完了") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestAddCmd_MissingConfig(t *testing.T) {
	cli := &CLI{
		UI: &addMockUI{
			confirms: []bool{false}, // 雛形作成をキャンセル
		},
		Config:     &addMockConfigStore{exists: false},
		TokenStore: &addMockTokenStore{},
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		NowFn:      time.Now,
	}

	cmd := NewAddCmd(cli)
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)

	err := cmd.Execute()
	if err == nil || err.Error() != "設定ファイルが存在しないため終了します" {
		t.Fatalf("expected abort error, got: %v", err)
	}
}

func TestAddCmd_AskTokenWhenNotSet(t *testing.T) {
	cfg := &Config{
		WorkspaceID:    "ws-1",
		Projects:       map[string]string{"dev": "111"},
		DefaultProject: "dev",
	}

	cli := &CLI{
		UI: &addMockUI{
			passwords: []string{"new-pat-from-input"},
			inputs:    []string{"テスト", "", "", "", ""},
		},
		Config:     &addMockConfigStore{exists: true, cfg: cfg},
		TokenStore: &addMockTokenStore{token: "", err: ErrTokenNotFound},
		Client:     &addMockAsanaClient{},
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		NowFn:      func() time.Time { return time.Time{} },
	}

	cmd := NewAddCmd(cli)
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 未登録時にトークンを保存したというメッセージが出ているか検証
	mockUI := cli.UI.(*addMockUI)
	output := strings.Join(mockUI.messages, "\n")
	if !strings.Contains(output, "トークンを保存しました") {
		t.Errorf("expected token save message, got: %s", output)
	}
}
