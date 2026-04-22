package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

// モックUI
type mockUI struct {
	inputs   []string
	inputIdx int
	confirms []bool
	confIdx  int
}

func (m *mockUI) Prompt(msg string, req bool) string {
	res := m.inputs[m.inputIdx]
	m.inputIdx++
	return res
}
func (m *mockUI) Show(msg string) {}
func (m *mockUI) Confirm(msg string) bool {
	res := m.confirms[m.confIdx]
	m.confIdx++
	return res
}

// モックConfig
type mockConfig struct {
	cfg    *Config
	exists bool
}

func (m *mockConfig) Exists() bool           { return m.exists }
func (m *mockConfig) Load() (*Config, error) { return m.cfg, nil }
func (m *mockConfig) CreateTemplate() error  { return nil }

// モックAPI
type mockClient struct{}

func (m *mockClient) CreateTask(ctx context.Context, task TaskData) (string, error) {
	return "https://asana.com/task/1", nil
}

func TestAppRun_SuccessFlow(t *testing.T) {
	cfg := &Config{
		PersonalAccessToken: "test-pat",
		WorkspaceID:         "ws-1",
		ProjectID:           "proj-1",
		Assignees:           map[string]string{"me": "123"},
	}

	app := &App{
		ui:     &mockUI{inputs: []string{"テストタスク", "me", "説明", "today"}},
		client: &mockClient{},
		config: &mockConfig{exists: true, cfg: cfg},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		nowFn:  func() time.Time { return time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC) },
	}

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppRun_MissingConfig(t *testing.T) {
	app := &App{
		ui:     &mockUI{confirms: []bool{false}}, // 作成をキャンセル
		config: &mockConfig{exists: false},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	err := app.Run(context.Background())
	if err == nil || err.Error() != "設定ファイルが存在しないため終了します" {
		t.Fatalf("expected abort error, got: %v", err)
	}
}
