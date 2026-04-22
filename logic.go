package main

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrAssigneeNotFound  = errors.New("assignee not found")
	ErrInvalidDateFormat = errors.New("invalid date format")
)

// ResolveAssignee は入力文字列からGIDを解決する純粋関数です（副作用なし）
func ResolveAssignee(input string, assignees map[string]string) (string, error) {
	if input == "" {
		return "", nil
	}
	if gid, ok := assignees[input]; ok {
		return gid, nil
	}
	return "", ErrAssigneeNotFound
}

// ResolveDueOn は日付入力（today等）を標準フォーマットに変換する純粋関数です
func ResolveDueOn(input string, now time.Time) (string, error) {
	if input == "" {
		return "", nil
	}
	if strings.ToLower(input) == "today" {
		return now.Format("2006-01-02"), nil
	}
	if _, err := time.Parse("2006-01-02", input); err != nil {
		return "", ErrInvalidDateFormat
	}
	return input, nil
}
