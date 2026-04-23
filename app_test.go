package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

// --- モックの実装 ---

type mockUI struct {
	inputs      []string
	inputIdx    int
	passwords   []string
	passwordIdx int
	confirms    []bool
	confIdx     int
}

func (m *mockUI) Prompt(msg string, req bool) string {
	if m.inputIdx >= len(m.inputs) {
		return ""
	}
	res := m.inputs[m.inputIdx]
	m.inputIdx++
	return res
}
func (m *mockUI) PromptPassword(msg string) string {
	if m.passwordIdx >= len(m.passwords) {
		return ""
	}
	res := m.passwords[m.passwordIdx]
	m.passwordIdx++
	return res
}
func (m *mockUI) Show(msg string) {}
func (m *mockUI) Confirm(msg string) bool {
	if m.confIdx >= len(m.confirms) {
		return false
	}
	res := m.confirms[m.confIdx]
	m.confIdx++
	return res
}

type mockConfig struct {
	cfg    *Config
	exists bool
}

func (m *mockConfig) Exists() bool           { return m.exists }
func (m *mockConfig) Load() (*Config, error) { return m.cfg, nil }
func (m *mockConfig) CreateTemplate() error  { return nil }

type mockClient struct{}

func (m *mockClient) CreateTask(ctx context.Context, task TaskData) (string, error) {
	return "https://asana.com/task/1", nil
}

type mockTokenStore struct {
	token string
	err   error
}

func (m *mockTokenStore) Get() (string, error)   { return m.token, m.err }
func (m *mockTokenStore) Set(token string) error { m.token = token; return nil }
func (m *mockTokenStore) Delete() error          { m.token = ""; return nil }

// --- Tests ---

func TestAppRun_SuccessFlow_WithExistingTokenAndDefaultProject(t *testing.T) {
	cfg := &Config{
		WorkspaceID:    "ws-1",
		Projects:       map[string]string{"dev": "111", "mktg": "222"},
		DefaultProject: "dev",
		Assignees:      map[string]string{"me": "123"},
	}

	app := &App{
		// 入力順: タスク名, プロジェクト(空でデフォルト), 担当者, 説明, 期日
		ui:         &mockUI{inputs: []string{"テストタスク", "", "me", "説明", "today"}},
		client:     &mockClient{},
		config:     &mockConfig{exists: true, cfg: cfg},
		tokenStore: &mockTokenStore{token: "existing-pat"},
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		nowFn:      func() time.Time { return time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC) },
	}

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppRun_AskTokenWhenNotSet(t *testing.T) {
	cfg := &Config{
		WorkspaceID:    "ws-1",
		Projects:       map[string]string{"dev": "111"},
		DefaultProject: "dev",
	}

	app := &App{
		ui: &mockUI{
			passwords: []string{"new-pat-from-input"},
			inputs:    []string{"テスト", "", "", "", ""},
		},
		client:     &mockClient{},
		config:     &mockConfig{exists: true, cfg: cfg},
		tokenStore: &mockTokenStore{token: "", err: ErrTokenNotFound},
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		nowFn:      func() time.Time { return time.Time{} },
	}

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppRun_MissingConfig(t *testing.T) {
	app := &App{
		ui:         &mockUI{confirms: []bool{false}},
		config:     &mockConfig{exists: false},
		tokenStore: &mockTokenStore{},
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		nowFn:      time.Now,
	}

	err := app.Run(context.Background())
	if err == nil || err.Error() != "設定ファイルが存在しないため終了します" {
		t.Fatalf("expected abort error, got: %v", err)
	}
}
