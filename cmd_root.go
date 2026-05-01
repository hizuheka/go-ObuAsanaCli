package main

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
)

type CLI struct {
	UI         UI
	Config     ConfigStore
	TokenStore TokenStore
	Client     AsanaClient // 追加: テスト用モックの注入ポイント
	Logger     *slog.Logger
	NowFn      func() time.Time
}

func NewRootCmd(cli *CLI) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "asanacli",
		Short: "Asana CLI Task Manager",
		Long:  "コマンドラインからAsanaのタスクを高速に管理するツールです。\n\n利用可能なコマンド:\n  asanacli add   (タスクを対話的に登録)\n  asanacli list  (タスクの一覧を表示)",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand(NewAddCmd(cli))
	rootCmd.AddCommand(NewListCmd(cli))

	return rootCmd
}

// setupConfigAndClient は各コマンド実行前に依存を解決します。
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
			cli.UI.Show("⚠️ 警告: トークンの保存に失敗しました。")
		} else {
			cli.UI.Show("✅ トークンを保存しました！")
		}
	}

	// 変更点: 既にClientが注入されている場合（テスト時）はそれを使う
	if cli.Client != nil {
		return cfg, cli.Client, nil
	}

	client := NewAsanaClient(pat)
	return cfg, client, nil
}
