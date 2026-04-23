package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type App struct {
	ui         UI
	client     AsanaClient
	config     ConfigStore
	tokenStore TokenStore
	logger     *slog.Logger
	nowFn      func() time.Time
}

func (a *App) Run(ctx context.Context) error {
	a.ui.Show("🚀 Asana Task Register")
	a.ui.Show("-----------------------")

	// 1. 設定ファイルのチェックと作成
	if !a.config.Exists() {
		a.ui.Show("設定ファイルが見つかりません。")
		if !a.ui.Confirm("新しい設定ファイルの雛形を作成しますか？ (Y/n)") {
			return errors.New("設定ファイルが存在しないため終了します")
		}
		if err := a.config.CreateTemplate(); err != nil {
			return fmt.Errorf("雛形の作成に失敗しました: %w", err)
		}
		a.ui.Show("✅ 雛形を作成しました。編集して再度実行してください。")
		return nil
	}

	// 2. ワークスペース等の設定読み込み
	cfg, err := a.config.Load()
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}
	if cfg.WorkspaceID == "" {
		return errors.New("設定ファイルの必須項目(WorkspaceID)が未入力です")
	}

	// 3. PAT(Personal Access Token)の取得（OSのシークレットストアから）
	pat, err := a.tokenStore.Get()
	if err != nil || pat == "" {
		a.ui.Show("⚠️ Personal Access Token がOSのシークレットに登録されていません。")
		pat = a.ui.PromptPassword("Asana PATを入力してください (入力は非表示になります): ")

		if err := a.tokenStore.Set(pat); err != nil {
			a.logger.Error("failed to save token to keyring", slog.Any("error", err))
			a.ui.Show("⚠️ 警告: トークンの安全な保存に失敗しました。次回も入力が必要になります。")
		} else {
			a.ui.Show("✅ トークンをシステムの資格情報に安全に保存しました！")
		}
	}

	// APIクライアントの生成
	if a.client == nil {
		a.client = NewAsanaClient(pat)
	}

	// 4. タスク名入力
	name := a.ui.Prompt("タスク名を入力してください: ", true)

	// 5. プロジェクト入力
	var projectGID string
	for {
		promptMsg := fmt.Sprintf("プロジェクトを入力してください (設定名 / 省略時は '%s'): ", cfg.DefaultProject)
		input := a.ui.Prompt(promptMsg, false)
		gid, err := ResolveProject(input, cfg.Projects, cfg.DefaultProject)
		if err != nil {
			a.ui.Show(fmt.Sprintf("⚠️ %v", err))
			continue
		}
		projectGID = gid

		targetName := input
		if input == "" {
			targetName = cfg.DefaultProject
		}
		a.ui.Show(fmt.Sprintf("  -> プロジェクトを '%s' に設定しました", targetName))
		break
	}

	// 6. 担当者入力
	var assigneeGID string
	for {
		input := a.ui.Prompt("担当者を入力してください (me / 設定名 / 省略可): ", false)
		gid, err := ResolveAssignee(input, cfg.Assignees)
		if err != nil {
			a.ui.Show(fmt.Sprintf("⚠️ '%s' は設定ファイルに登録されていません。", input))
			continue
		}
		assigneeGID = gid
		break
	}

	// 7. タスクの説明入力
	notes := a.ui.Prompt("タスクの説明を入力してください (省略可): ", false)

	// 8. 期日入力
	var dueOn string
	for {
		input := a.ui.Prompt("期日を入力してください (today / YYYY-MM-DD / 省略可): ", false)
		resolved, err := ResolveDueOn(input, a.nowFn())
		if err != nil {
			a.ui.Show("❌ エラー: 日付形式が正しくありません。")
			continue
		}
		dueOn = resolved
		if strings.ToLower(input) == "today" {
			a.ui.Show(fmt.Sprintf("  -> 期日を %s に設定しました", dueOn))
		}
		break
	}

	// 9. APIリクエスト
	a.ui.Show("\n📡 Asanaに登録中...")
	taskData := TaskData{
		Name:      name,
		Workspace: cfg.WorkspaceID,
		Projects:  []string{projectGID},
		Notes:     notes,
		Assignee:  assigneeGID,
		DueOn:     dueOn,
	}

	url, err := a.client.CreateTask(ctx, taskData)
	if err != nil {
		a.logger.Error("API Error", slog.Any("error", err))
		return fmt.Errorf("通信エラー: %w", err)
	}

	a.ui.Show("✅ 登録完了！\n🔗 " + url)
	return nil
}
