package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// CommandRunner は外部コマンドの実行を抽象化するインターフェースです。
// テスト時にはモックに差し替えることで、実際にコマンドを起動せずにテストが可能になります。
type CommandRunner interface {
	// RunInteractive は指定されたコマンドを実行し、標準入力にデータを流し込み、
	// ユーザーの対話的な操作結果（標準出力）を取得します。
	RunInteractive(ctx context.Context, command string, args []string, input []byte) ([]byte, error)
}

// execCommandRunner は CommandRunner インターフェースの標準的な実装（本番用）です。
type execCommandRunner struct{}

// NewCommandRunner は新しい CommandRunner を生成します。
func NewCommandRunner() CommandRunner {
	return &execCommandRunner{}
}

// RunInteractive は os/exec パッケージを使用して外部コマンドを実行します。
func (r *execCommandRunner) RunInteractive(ctx context.Context, command string, args []string, input []byte) ([]byte, error) {
	// コンテキスト付きでコマンドを生成（タイムアウトやキャンセルに対応可能）
	cmd := exec.CommandContext(ctx, command, args...)

	// fzf のような対話型CLIツールは標準エラー出力(Stderr)や特定のデバイスファイル(tty)
	// にUIを描画するため、現在のプロセスの Stderr をそのまま接続します。
	cmd.Stderr = os.Stderr

	// データをパイプや一時ファイルではなく、直接標準入力ストリームとして流し込みます。
	// これにより、WindowsのPowerShell等で発生するパイプラインの文字コード変換（CP932化）
	// をバイパスし、純粋なUTF-8バイト列としてデータを渡すことができます。
	cmd.Stdin = bytes.NewReader(input)

	// コマンドを実行し、標準出力の結果を取得します。
	// fzf の場合、ユーザーが選択して Enter を押した行がここに入ります。
	output, err := cmd.Output()
	if err != nil {
		// exec.ExitError はコマンドが非ゼロのステータスで終了した場合（例: fzfでESCキー押下）に返されます。
		// この場合、呼び出し元でキャンセルとして扱えるよう、エラーをラップして返します。
		return nil, fmt.Errorf("コマンド '%s' の実行に失敗、またはキャンセルされました: %w", command, err)
	}

	return output, nil
}
