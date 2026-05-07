package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/inconshreveable/mousetrap"
)

func main() {
	// 標準の構造化ロガー（slog）を初期化
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	// 設定ストアの初期化
	configStore, err := NewYamlConfigStore()
	if err != nil {
		logger.Error("Failed to initialize config store", slog.Any("error", err))
		os.Stdout.WriteString("❌ システムエラー: 設定ファイルの初期化に失敗しました。\n")
		waitBeforeExit()
		os.Exit(1)
	}

	// 依存性注入（DI）コンテナとしてCLI構造体を組み立てる
	// これにより、サブコマンド群はグローバル変数に依存せずテスト可能になります
	cli := &CLI{
		UI:         NewConsoleUI(os.Stdin, os.Stdout),
		Config:     configStore,
		TokenStore: NewTokenStore(),
		Logger:     logger,
		NowFn:      time.Now,
	}

	// ルートコマンドの生成
	rootCmd := NewRootCmd(cli)

	// Cobraの標準出力/標準エラー出力を制御
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	// コマンドの実行
	if err := rootCmd.Execute(); err != nil {
		// Cobraが自動的にエラーメッセージを出力するため、ここでは待機と終了コードの返却のみ行う
		waitBeforeExit()
		os.Exit(1)
	}

	// 正常終了時、エクスプローラー等からのダブルクリック起動を考慮して待機
	waitBeforeExit()
}

// waitBeforeExit は、エクスプローラーから直接起動された場合のみ待機します
func waitBeforeExit() {
	// StartedByExplorer() は、Windows環境でexeがダブルクリックで
	// 起動された場合にのみ true を返します。
	// コマンドラインからの実行や、"| fzf" のようなパイプ実行時は false となるため、
	// 待機処理が発生せず、他のコマンドとスムーズに連携できます。
	if mousetrap.StartedByExplorer() {
		fmt.Print("\nエンターキーを押して終了してください...")
		bufio.NewScanner(os.Stdin).Scan()
	}
}
