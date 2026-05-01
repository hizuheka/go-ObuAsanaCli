package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewAddCmd(cli *CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "対話形式で新しいタスクを登録します",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli.UI.Show("🚀 Asana Task Register")
			cli.UI.Show("-----------------------")

			cfg, client, err := setupConfigAndClient(cli)
			if err != nil {
				return err
			}

			name := cli.UI.Prompt("タスク名を入力してください: ", true)

			var projectGID string
			for {
				promptMsg := fmt.Sprintf("プロジェクトを入力してください (設定名 / 省略時は '%s'): ", cfg.DefaultProject)
				input := cli.UI.Prompt(promptMsg, false)
				gid, err := ResolveProject(input, cfg.Projects, cfg.DefaultProject)
				if err != nil {
					cli.UI.Show(fmt.Sprintf("⚠️ %v", err))
					continue
				}
				projectGID = gid
				break
			}

			var assigneeGID string
			for {
				input := cli.UI.Prompt("担当者を入力してください (me / 設定名 / 省略可): ", false)
				gid, err := ResolveAssignee(input, cfg.Assignees)
				if err != nil {
					cli.UI.Show(fmt.Sprintf("⚠️ '%s' は設定ファイルに登録されていません。", input))
					continue
				}
				assigneeGID = gid
				break
			}

			notes := cli.UI.Prompt("タスクの説明を入力してください (省略可): ", false)

			var dueOn string
			for {
				input := cli.UI.Prompt("期日を入力してください (today / YYYY-MM-DD / 省略可): ", false)
				resolved, err := ResolveDueOn(input, cli.NowFn())
				if err != nil {
					cli.UI.Show("❌ エラー: 日付形式が正しくありません。")
					continue
				}
				dueOn = resolved
				break
			}

			cli.UI.Show("\n📡 Asanaに登録中...")
			taskData := TaskData{
				Name:      name,
				Workspace: cfg.WorkspaceID,
				Projects:  []string{projectGID},
				Notes:     notes,
				Assignee:  assigneeGID,
				DueOn:     dueOn,
			}

			url, err := client.CreateTask(cmd.Context(), taskData)
			if err != nil {
				return fmt.Errorf("通信エラー: %w", err)
			}

			cli.UI.Show("✅ 登録完了！\n🔗 " + url)
			return nil
		},
	}
}
