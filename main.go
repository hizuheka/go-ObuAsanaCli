package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	configStore, err := NewYamlConfigStore()
	if err != nil {
		fmt.Printf("❌ システムエラー: %v\n", err)
		os.Exit(1)
	}

	// 依存オブジェクトの組み立て
	app := &App{
		ui:         NewConsoleUI(os.Stdin, os.Stdout),
		config:     configStore,
		tokenStore: NewTokenStore(), // OSの資格情報マネージャーを注入
		logger:     logger,
		client:     nil, // トークン取得後に生成するため初期はnil
		nowFn:      time.Now,
	}

	if err := app.Run(context.Background()); err != nil {
		fmt.Printf("\n❌ %v\n", err)
		os.Exit(1)
	}
}
