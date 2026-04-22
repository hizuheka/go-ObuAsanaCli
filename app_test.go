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
	inputs   []string
	inputIdx int
	confirms []bool
	confIdx  int
}

func (m *mockUI) Prompt(msg string, req bool) string {
	if m.inputIdx >= len(m.inputs) {
		return "" // パニック防止
	}
	res := m.inputs[m.inputIdx]
	m.inputIdx++
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

// --- テストケース ---

// デフォルトプロジェクトが適用されるフロー（プロジェクト入力を空打ちした場合）
func TestAppRun_SuccessFlow_WithDefaultProject(t *testing.T) {
	cfg := &Config{
		PersonalAccessToken: "test-pat",
		WorkspaceID:         "ws-1",
		Projects:            map[string]string{"dev": "111", "mktg": "222"},
		DefaultProject:      "dev", // デフォルト指定
		Assignees:           map[string]string{"me": "123"},
	}

	app := &App{
		// 入力順: タスク名, プロジェクト, 担当者, 説明, 期日
		// 2番目の "" がプロジェクト入力のスキップ（デフォルト適用）をエミュレート
		ui:     &mockUI{inputs: []string{"テストタスク", "", "me", "説明", "today"}},
		client: &mockClient{},
		config: &mockConfig{exists: true, cfg: cfg},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		nowFn:  func() time.Time { return time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC) },
	}

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// 特定のプロジェクトを明示的に指定するフロー
func TestAppRun_SuccessFlow_WithSpecificProject(t *testing.T) {
	cfg := &Config{
		PersonalAccessToken: "test-pat",
		WorkspaceID:         "ws-1",
		Projects:            map[string]string{"dev": "111", "mktg": "222"},
		DefaultProject:      "dev",
		Assignees:           map[string]string{"me": "123"},
	}

	app := &App{
		// 入力順: タスク名, プロジェクト, 担当者, 説明, 期日
		// "mktg" を明示的に指定
		ui:     &mockUI{inputs: []string{"マーケティングタスク", "mktg", "me", "", "2026-12-31"}},
		client: &mockClient{},
		config: &mockConfig{exists: true, cfg: cfg},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		nowFn:  func() time.Time { return time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC) },
	}

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// 設定ファイルが存在せず、作成もキャンセルした場合
func TestAppRun_MissingConfig(t *testing.T) {
	app := &App{
		ui:     &mockUI{confirms: []bool{false}}, // 作成をキャンセル
		config: &mockConfig{exists: false},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		nowFn:  time.Now,
	}

	err := app.Run(context.Background())
	if err == nil || err.Error() != "設定ファイルが存在しないため終了します" {
		t.Fatalf("expected abort error, got: %v", err)
	}
}
