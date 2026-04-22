package main

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrAssigneeNotFound  = errors.New("assignee not found")
	ErrInvalidDateFormat = errors.New("invalid date format")
	ErrProjectNotFound   = errors.New("project not found")
	ErrDefaultNotSet     = errors.New("default project is not set")
)

// ResolveProject はプロジェクトの入力を解決する純粋関数です
func ResolveProject(input string, projects map[string]string, defaultProj string) (string, error) {
	target := input
	// 入力が空ならデフォルトプロジェクトを採用
	if target == "" {
		if defaultProj == "" {
			return "", ErrDefaultNotSet
		}
		target = defaultProj
	}

	if gid, ok := projects[target]; ok {
		return gid, nil
	}
	return "", ErrProjectNotFound
}

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
