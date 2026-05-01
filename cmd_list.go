package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type ListOptions struct {
	Project  string
	Status   string // open, closed, all
	Assignee string // config.yamlに定義したエイリアス (例: me)
	Output   string // text, json
}

func NewListCmd(cli *CLI) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "プロジェクトのタスク一覧を表示・フィルタリングします",
		Example: `  asanacli list
  asanacli list --status closed
  asanacli list --project mktg --assignee me
  asanacli list --output json | jq '.'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, client, err := setupConfigAndClient(cli)
			if err != nil {
				return err
			}

			projectGID, err := ResolveProject(opts.Project, cfg.Projects, cfg.DefaultProject)
			if err != nil {
				return fmt.Errorf("プロジェクトの解決に失敗しました: %w", err)
			}

			var assigneeGID string
			if opts.Assignee != "" {
				assigneeGID, err = ResolveAssignee(opts.Assignee, cfg.Assignees)
				if err != nil {
					return fmt.Errorf("担当者の解決に失敗しました: %w", err)
				}
			}

			if opts.Status != "open" && opts.Status != "closed" && opts.Status != "all" {
				return fmt.Errorf("--status は 'open', 'closed', 'all' のいずれかを指定してください")
			}

			if opts.Output != "json" {
				cli.UI.Show("📡 タスクを取得中...")
			}

			tasks, err := client.GetTasks(cmd.Context(), projectGID)
			if err != nil {
				return fmt.Errorf("タスク取得エラー: %w", err)
			}

			var filtered []TaskResponseData
			for _, task := range tasks {
				// ステータスフィルタ
				if opts.Status == "open" && task.Completed {
					continue
				}
				if opts.Status == "closed" && !task.Completed {
					continue
				}
				// 担当者フィルタ
				if assigneeGID != "" && task.Assignee() != assigneeGID {
					continue
				}
				filtered = append(filtered, task)
			}

			// JSON出力モード (fzf連携などスクリプト用)
			if opts.Output == "json" {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(filtered); err != nil {
					return fmt.Errorf("JSONの生成に失敗しました: %w", err)
				}
				return nil
			}

			// テキスト出力モード (人間用)
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "STATUS\tDUE DATE\tASSIGNEE\tTASK NAME")
			fmt.Fprintln(w, "------\t--------\t--------\t---------")

			for _, task := range filtered {
				status := "[ ]"
				if task.Completed {
					status = "[x]"
				}

				due := task.DueOn
				if due == "" {
					due = "未設定    "
				}

				assigneeDisp := "-"
				if task.Assignee() != "" {
					assigneeDisp = "Assigned"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", status, due, assigneeDisp, task.Name)
			}
			w.Flush()

			if len(filtered) == 0 {
				cli.UI.Show("\n条件に一致するタスクがありません。")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "表示するプロジェクト (省略時はデフォルト)")
	cmd.Flags().StringVarP(&opts.Status, "status", "s", "open", "ステータスで絞り込み (open, closed, all)")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "担当者で絞り込み (例: me)")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "text", "出力フォーマット (text, json)")

	return cmd
}
