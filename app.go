package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// App は依存オブジェクトを持ち、一連の実行フローを管理します
type App struct {
	ui     UI
	client AsanaClient
	config ConfigStore
	logger *slog.Logger
	nowFn  func() time.Time // モック化を容易にするための関数ポインタ
}

func (a *App) Run(ctx context.Context) error {
	a.ui.Show("🚀 Asana Task Register")
	a.ui.Show("-----------------------")

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

	cfg, err := a.config.Load()
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	if cfg.PersonalAccessToken == "" || cfg.WorkspaceID == "" || cfg.ProjectID == "" {
		return errors.New("設定ファイルの必須項目(PAT, WorkspaceID, ProjectID)が未入力です")
	}

	// 実行時までAPIクライアントが作られていない場合はここで遅延生成する
	if a.client == nil {
		a.client = NewAsanaClient(cfg.PersonalAccessToken)
	}

	name := a.ui.Prompt("タスク名を入力してください: ", true)

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

	notes := a.ui.Prompt("タスクの説明を入力してください (省略可): ", false)

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

	a.ui.Show("\n📡 Asanaに登録中...")
	taskData := TaskData{
		Name:      name,
		Workspace: cfg.WorkspaceID,
		Projects:  []string{cfg.ProjectID},
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
