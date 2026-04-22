package main

import (
	"errors"
	"testing"
	"time"
)

func TestResolveAssignee(t *testing.T) {
	assignees := map[string]string{"me": "123"}

	tests := []struct {
		input   string
		want    string
		wantErr error
	}{
		{"", "", nil},
		{"me", "123", nil},
		{"john", "", ErrAssigneeNotFound},
	}

	for _, tt := range tests {
		got, err := ResolveAssignee(tt.input, assignees)
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("input %q: want err %v, got %v", tt.input, tt.wantErr, err)
		}
		if got != tt.want {
			t.Errorf("input %q: want %v, got %v", tt.input, tt.want, got)
		}
	}
}

func TestResolveDueOn(t *testing.T) {
	now := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		input   string
		want    string
		wantErr error
	}{
		{"", "", nil},
		{"today", "2026-04-21", nil},
		{"TODAY", "2026-04-21", nil},
		{"2026-12-31", "2026-12-31", nil},
		{"2026/12/31", "", ErrInvalidDateFormat},
	}

	for _, tt := range tests {
		got, err := ResolveDueOn(tt.input, now)
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("input %q: want err %v, got %v", tt.input, tt.wantErr, err)
		}
		if got != tt.want {
			t.Errorf("input %q: want %v, got %v", tt.input, tt.want, got)
		}
	}
}

func TestResolveProject(t *testing.T) {
	projects := map[string]string{
		"dev":  "111",
		"mktg": "222",
	}

	tests := []struct {
		name        string
		input       string
		defaultProj string
		want        string
		wantErr     error
	}{
		{"直接指定（成功）", "mktg", "dev", "222", nil},
		{"空入力でデフォルト適用", "", "dev", "111", nil},
		{"存在しないプロジェクト", "sales", "dev", "", ErrProjectNotFound},
		{"デフォルト未設定で空入力", "", "", "", ErrDefaultNotSet},
		{"デフォルト設定のプロジェクトが存在しない", "", "wrong", "", ErrProjectNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProject(tt.input, projects, tt.defaultProj)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("want err %v, got %v", tt.wantErr, err)
			}
			if got != tt.want {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}
