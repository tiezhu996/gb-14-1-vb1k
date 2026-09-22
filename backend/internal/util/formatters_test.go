package util

import (
	"testing"

	"github.com/blueship581/codelearn/internal/constants"
)

func TestNormalizeOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trailing spaces", in: "3  \n 4\t\n", want: "3\n 4"},
		{name: "crlf", in: "3\r\n4\r\n", want: "3\n4"},
		{name: "empty", in: "", want: ""},
		{name: "plain", in: "fl", want: "fl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeOutput(tt.in); got != tt.want {
				t.Errorf("NormalizeOutput(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatDifficultyText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "easy", in: constants.DifficultyEasy, want: "简单"},
		{name: "medium", in: constants.DifficultyMedium, want: "中等"},
		{name: "hard", in: constants.DifficultyHard, want: "困难"},
		{name: "unknown", in: "unknown", want: "未知"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDifficultyText(tt.in); got != tt.want {
				t.Errorf("FormatDifficultyText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatSubmissionStatusText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "accepted", in: constants.SubmissionAccepted, want: "通过"},
		{name: "partial", in: constants.SubmissionPartial, want: "部分通过"},
		{name: "timeout", in: constants.SubmissionTimeout, want: "超时"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatSubmissionStatusText(tt.in); got != tt.want {
				t.Errorf("FormatSubmissionStatusText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatRoleText(t *testing.T) {
	if got := FormatRoleText(constants.RoleAdmin); got != "管理员" {
		t.Errorf("FormatRoleText(admin) = %q", got)
	}
	if got := FormatRoleText(constants.RoleStudent); got != "学生" {
		t.Errorf("FormatRoleText(student) = %q", got)
	}
}

func TestFormatRejudgeDiffText(t *testing.T) {
	tests := []struct {
		name       string
		prevStatus string
		newStatus  string
		scoreDelta int
		want       string
	}{
		{name: "失败转通过", prevStatus: constants.SubmissionRuntimeError, newStatus: constants.SubmissionAccepted, scoreDelta: 100, want: "运行错误 → 通过（得分 +100）"},
		{name: "部分通过转通过", prevStatus: constants.SubmissionPartial, newStatus: constants.SubmissionAccepted, scoreDelta: 50, want: "部分通过 → 通过（得分 +50）"},
		{name: "得分下降", prevStatus: constants.SubmissionPartial, newStatus: constants.SubmissionTimeout, scoreDelta: -50, want: "部分通过 → 超时（得分 -50）"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatRejudgeDiffText(tt.prevStatus, tt.newStatus, tt.scoreDelta); got != tt.want {
				t.Errorf("FormatRejudgeDiffText() = %q, want %q", got, tt.want)
			}
		})
	}
}
