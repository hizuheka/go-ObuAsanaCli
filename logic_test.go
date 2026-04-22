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
