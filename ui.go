package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// UI はユーザーとの対話を抽象化します。
// テスト時に入力をモック化するために必要です。
type UI interface {
	Prompt(message string, required bool) string
	PromptPassword(message string) string // 入力内容を隠蔽する
	Show(message string)
	Confirm(message string) bool
}

type consoleUI struct {
	scanner *bufio.Scanner
	out     io.Writer
}

func NewConsoleUI(in io.Reader, out io.Writer) UI {
	return &consoleUI{
		scanner: bufio.NewScanner(in),
		out:     out,
	}
}

func (u *consoleUI) Prompt(message string, required bool) string {
	for {
		fmt.Fprint(u.out, message)
		if !u.scanner.Scan() {
			return ""
		}
		input := strings.TrimSpace(u.scanner.Text())
		if required && input == "" {
			fmt.Fprintln(u.out, "❌ この項目は必須です。")
			continue
		}
		return input
	}
}

func (u *consoleUI) PromptPassword(message string) string {
	for {
		fmt.Fprint(u.out, message)
		// os.Stdinのファイルディスクリプタを取得してエコーバックを無効化
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(u.out) // 入力後に改行を入れる
		if err != nil {
			return ""
		}
		input := strings.TrimSpace(string(bytePassword))
		if input == "" {
			fmt.Fprintln(u.out, "❌ トークンは必須です。")
			continue
		}
		return input
	}
}

func (u *consoleUI) Show(message string) {
	fmt.Fprintln(u.out, message)
}

func (u *consoleUI) Confirm(message string) bool {
	fmt.Fprint(u.out, message+" ")
	if !u.scanner.Scan() {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(u.scanner.Text()))
	return ans == "" || ans == "y" || ans == "yes"
}
