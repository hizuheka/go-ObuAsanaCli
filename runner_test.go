package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestHelperProcess は外部コマンドのモックとして振る舞うための特殊なテスト関数です。
// テストプロセス自身をサブプロセスとして起動した際に、特定の環境変数がある場合のみ実行され、
// モックとして振る舞った後に os.Exit で即座に終了します。
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// 異常系のテスト用: 非ゼロステータスで終了
	if os.Getenv("GO_HELPER_EXIT_ERROR") == "1" {
		os.Exit(1)
	}

	// コンテキストキャンセルのテスト用: 長時間スリープ
	if os.Getenv("GO_HELPER_SLEEP") == "1" {
		time.Sleep(10 * time.Second)
		os.Exit(0)
	}

	// 正常系のテスト用: 標準入力をそのまま標準出力へエコー
	_, _ = io.Copy(os.Stdout, os.Stdin)
	os.Exit(0)
}

func TestExecCommandRunner_RunInteractive_Success(t *testing.T) {
	runner := NewCommandRunner()
	ctx := context.Background()

	inputData := []byte("12345\t[ ] Selected Task\n")

	// os.Args[0] はテスト実行時のバイナリパスを指します。
	// これを利用して、自分自身をサブプロセスとして起動します。
	cmd := os.Args[0]
	args := []string{"-test.run=TestHelperProcess", "--"}

	// サブプロセス側でモックとして振る舞うためのフラグを設定
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	output, err := runner.RunInteractive(ctx, cmd, args, inputData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(output) != string(inputData) {
		t.Errorf("expected output %q, got %q", string(inputData), string(output))
	}
}

func TestExecCommandRunner_RunInteractive_ExitError(t *testing.T) {
	runner := NewCommandRunner()
	ctx := context.Background()

	cmd := os.Args[0]
	args := []string{"-test.run=TestHelperProcess", "--"}

	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	// 異常終了モードを有効化
	t.Setenv("GO_HELPER_EXIT_ERROR", "1")

	_, err := runner.RunInteractive(ctx, cmd, args, nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	expectedPrefix := fmt.Sprintf("コマンド '%s' の実行に失敗、またはキャンセルされました:", cmd)
	if !strings.HasPrefix(err.Error(), expectedPrefix) {
		t.Errorf("expected error to start with %q, got %q", expectedPrefix, err.Error())
	}
}

func TestExecCommandRunner_RunInteractive_ContextCanceled(t *testing.T) {
	runner := NewCommandRunner()

	// 即座にキャンセルされるコンテキストを作成
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := os.Args[0]
	args := []string{"-test.run=TestHelperProcess", "--"}

	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	// スリープモードを有効化し、プロセスの完了を待機させる
	t.Setenv("GO_HELPER_SLEEP", "1")

	_, err := runner.RunInteractive(ctx, cmd, args, nil)
	if err == nil {
		t.Fatal("expected an error due to context cancellation, got nil")
	}

	expectedPrefix := fmt.Sprintf("コマンド '%s' の実行に失敗、またはキャンセルされました:", cmd)
	if !strings.HasPrefix(err.Error(), expectedPrefix) {
		t.Errorf("expected cancellation error message, got %q", err.Error())
	}
}
