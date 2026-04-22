package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

func main() {
	// ログは標準エラー出力へ（UIと分離）
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	// 設定ストアの初期化
	configStore, err := NewYamlConfigStore()
	if err != nil {
		fmt.Printf("❌ システムエラー: %v\n", err)
		os.Exit(1)
	}

	// 依存オブジェクトの組み立て (DI)
	app := &App{
		ui:     NewConsoleUI(os.Stdin, os.Stdout),
		config: configStore,
		logger: logger,
		// APIクライアントは設定ファイル読み込み後に app.go 内で生成する
		client: nil,
		nowFn:  time.Now,
	}

	// 実行
	if err := app.Run(context.Background()); err != nil {
		fmt.Printf("\n❌ %v\n", err)
		os.Exit(1)
	}
}
