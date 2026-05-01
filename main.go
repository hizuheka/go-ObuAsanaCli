package main

import (
	"bufio"
	"log/slog"
	"os"
	"time"
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

// waitBeforeExit は、プログラム終了前にユーザーがエンターキーを押すまで待機します。
// エクスプローラーから直接実行した際に、結果の出力が閉じて見えなくなるのを防ぎます。
func waitBeforeExit() {
	// 注: JSON出力時など、パイプ（|）を通じて他のプログラムにデータを渡している最中は
	// 待機プロンプトを出さないための処理を将来的に追加する余地があります。
	// 今回は確実性を優先し、常に待機します。
	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) != 0 {
		os.Stdout.WriteString("\nエンターキーを押して終了してください...")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
	}
}
