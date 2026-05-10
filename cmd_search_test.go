package main

import (
	"context"
	"strings"
	"testing"
)

// --- 純粋関数のテスト ---

func TestFormatTasksForSearch(t *testing.T) {
	tasks := []TaskResponseData{
		{GID: "101", Name: "Incomplete Task", Completed: false},
		{GID: "102", Name: "Completed Task", Completed: true},
	}

	got := formatTasksForSearch(tasks)
	expected := "101\t[ ] Incomplete Task\n102\t[x] Completed Task\n"

	if string(got) != expected {
		t.Errorf("expected %q, got %q", expected, string(got))
	}
}

func TestFormatTasksForSearch_Empty(t *testing.T) {
	got := formatTasksForSearch([]TaskResponseData{})
	if string(got) != "" {
		t.Errorf("expected empty string, got %q", string(got))
	}
}

func TestExtractGIDFromSearchOutput(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"正常系_標準的なfzf出力", []byte("12345\t[ ] My Task\n"), "12345"},
		{"正常系_改行なし", []byte("67890\t[x] Another Task"), "67890"},
		{"異常系_タブなし(不正フォーマット)", []byte("JustString"), "JustString"},
		{"異常系_空文字", []byte("   \n"), ""},
		{"異常系_nil入力", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractGIDFromSearchOutput(tt.input); got != tt.want {
				t.Errorf("extractGIDFromSearchOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- コマンドのテスト ---

// mockCommandRunner は外部コマンド呼び出しをバイパスするモックです。
type mockCommandRunner struct {
	output []byte
	err    error
	called bool
}

func (m *mockCommandRunner) RunInteractive(ctx context.Context, command string, args []string, input []byte) ([]byte, error) {
	m.called = true
	return m.output, m.err
}

func TestSearchCmd_Success(t *testing.T) {
	tasks := []TaskResponseData{
		{GID: "999", Name: "Target Task", Completed: false},
	}

	// 以前作成したセットアップ関数を流用（Config/Clientのモックが含まれている前提）
	cli, outBuf := setupTestCLI(tasks)

	// ユーザーがfzf上でタスクを選択し、Enterを押した状態をエミュレート
	mockRunner := &mockCommandRunner{
		output: []byte("999\t[ ] Target Task\n"),
		err:    nil,
	}
	cli.Runner = mockRunner

	cmd := NewSearchCmd(cli)
	cmd.SetOut(outBuf)
	cmd.SetArgs([]string{}) // デフォルトオプション

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mockRunner.called {
		t.Error("expected CommandRunner to be called")
	}

	// 標準出力にGIDだけが出力されているか検証
	output := strings.TrimSpace(outBuf.String())
	if output != "999" {
		t.Errorf("expected GID '999' in output, got %q", output)
	}
}

func TestSearchCmd_EmptyResult(t *testing.T) {
	// 完了済みタスクのみが存在する状況
	tasks := []TaskResponseData{
		{GID: "888", Name: "Done Task", Completed: true},
	}

	cli, outBuf := setupTestCLI(tasks)

	mockRunner := &mockCommandRunner{}
	cli.Runner = mockRunner

	cmd := NewSearchCmd(cli)
	cmd.SetOut(outBuf)
	// 未完了(open)を検索するため、結果は0件になるはず
	cmd.SetArgs([]string{"--status", "open"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockRunner.called {
		t.Error("expected CommandRunner NOT to be called when no tasks found")
	}
}
