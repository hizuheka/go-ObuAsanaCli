package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// SearchOptions は search コマンドのフラグを保持します。
type SearchOptions struct {
	Project string
	Status  string
}

// formatTasksForSearch はタスクのリストをfzfに渡すためのタブ区切り文字列に変換する純粋関数です。
// C1カバレッジ網羅のため、完了/未完了の分岐を含みます。
func formatTasksForSearch(tasks []TaskResponseData) []byte {
	var buf bytes.Buffer
	for _, t := range tasks {
		status := "[ ]"
		if t.Completed {
			status = "[x]"
		}
		buf.WriteString(fmt.Sprintf("%s\t%s %s\n", t.GID, status, t.Name))
	}
	return buf.Bytes()
}

// extractGIDFromSearchOutput はfzfの出力からGID部分のみを抽出する純粋関数です。
func extractGIDFromSearchOutput(output []byte) string {
	str := strings.TrimSpace(string(output))
	if str == "" {
		return ""
	}

	// タブで分割し、1つ目の要素（GID）を取得
	parts := strings.SplitN(str, "\t", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// NewSearchCmd は search サブコマンドを生成します。
func NewSearchCmd(cli *CLI) *cobra.Command {
	opts := &SearchOptions{}

	cmd := &cobra.Command{
		Use:   "search",
		Short: "インタラクティブなUIでタスクを検索し、GIDを取得します",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, client, err := setupConfigAndClient(cli)
			if err != nil {
				return err
			}

			projectGID, err := ResolveProject(opts.Project, cfg.Projects, cfg.DefaultProject)
			if err != nil {
				return fmt.Errorf("プロジェクト解決エラー: %w", err)
			}

			cli.UI.Show("📡 タスクを取得中...")
			tasks, err := client.GetTasks(cmd.Context(), projectGID)
			if err != nil {
				return fmt.Errorf("タスク取得エラー: %w", err)
			}

			var filtered []TaskResponseData
			for _, task := range tasks {
				if opts.Status == "open" && task.Completed {
					continue
				}
				if opts.Status == "closed" && !task.Completed {
					continue
				}
				filtered = append(filtered, task)
			}

			if len(filtered) == 0 {
				cli.UI.Show("条件に一致するタスクがありません。")
				return nil
			}

			inputData := formatTasksForSearch(filtered)

			// fzfの引数: タブ区切りとし、2列目以降（チェックボックスとタスク名）のみを画面に表示
			fzfArgs := []string{"--delimiter=\\t", "--with-nth=2..", "--prompt=Select Task> "}

			// OSのシェルパイプを使わずにGoプロセスから直接fzfを起動・通信
			output, err := cli.Runner.RunInteractive(cmd.Context(), "fzf", fzfArgs, inputData)
			if err != nil {
				// fzfでESCキーを押した（キャンセル）場合などもエラーとなるためログのみ記録
				cli.Logger.Info("fzf execution finished or canceled", "error", err.Error())
				return nil
			}

			selectedGID := extractGIDFromSearchOutput(output)
			if selectedGID != "" {
				// 標準出力にGIDだけを書き出す（他のコマンドとの連携用）
				fmt.Fprintln(cmd.OutOrStdout(), selectedGID)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "検索対象のプロジェクト (省略時はデフォルト)")
	cmd.Flags().StringVarP(&opts.Status, "status", "s", "open", "ステータスで絞り込み (open, closed, all)")

	return cmd
}
