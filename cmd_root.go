package main

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
)

// CLI はすべてのコマンドで共有される依存関係（DIコンテナ）です。
type CLI struct {
	UI         UI
	Config     ConfigStore
	TokenStore TokenStore
	Client     AsanaClient
	Runner     CommandRunner
	Logger     *slog.Logger
	NowFn      func() time.Time
}

// NewRootCmd はCobraのルートコマンドを初期化します。
func NewRootCmd(cli *CLI) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "asanacli",
		Short: "Asana CLI Task Manager",
		Long:  "コマンドラインからAsanaのタスクを高速に管理するツールです。\n\n利用可能なコマンド:\n  asanacli add   (タスクを対話的に登録)\n  asanacli list  (タスクの一覧を表示)",
		// 引数なしで実行された場合は、デフォルトとして 'add' (タスク登録) を実行する
		RunE: func(cmd *cobra.Command, args []string) error {
			addCmd := NewAddCmd(cli)
			return addCmd.RunE(cmd, args)
		},
	}

	// サブコマンドを登録
	rootCmd.AddCommand(NewAddCmd(cli))
	rootCmd.AddCommand(NewListCmd(cli))
	rootCmd.AddCommand(NewSearchCmd(cli))

	return rootCmd
}

// setupConfigAndClient は各コマンドが実行される前に呼び出される共通の初期化関数です。
// 設定ファイルの読み込みと、PATの取得（およびAsanaClientの生成）を行います。
func setupConfigAndClient(cli *CLI) (*Config, AsanaClient, error) {
	if !cli.Config.Exists() {
		cli.UI.Show("設定ファイルが見つかりません。")
		if !cli.UI.Confirm("新しい設定ファイルの雛形を作成しますか？ (Y/n)") {
			return nil, nil, errors.New("設定ファイルが存在しないため終了します")
		}
		if err := cli.Config.CreateTemplate(); err != nil {
			return nil, nil, fmt.Errorf("雛形の作成に失敗しました: %w", err)
		}
		cli.UI.Show("✅ 雛形を作成しました。編集して再度実行してください。")
		// 雛形を作った直後は終了する
		return nil, nil, errors.New("初期セットアップ完了")
	}

	cfg, err := cli.Config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	pat, err := cli.TokenStore.Get()
	if err != nil || pat == "" {
		cli.UI.Show("⚠️ Personal Access Token がOSのシークレットに登録されていません。")
		pat = cli.UI.PromptPassword("Asana PATを入力してください (入力は非表示になります): ")
		if err := cli.TokenStore.Set(pat); err != nil {
			cli.Logger.Error("failed to save token", slog.Any("error", err))
			cli.UI.Show("⚠️ 警告: トークンの保存に失敗しました。")
		} else {
			cli.UI.Show("✅ トークンを保存しました！")
		}
	}

	// テスト時など、すでにClientが注入されている場合はそれを使用する
	if cli.Client != nil {
		return cfg, cli.Client, nil
	}

	client := NewAsanaClient(pat)
	return cfg, client, nil
}
