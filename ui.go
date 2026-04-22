package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// UI はユーザーとの対話を抽象化します。
// テスト時に入力をモック化するために必要です。
type UI interface {
	Prompt(message string, required bool) string
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
