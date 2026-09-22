package constants

import "testing"

func TestRejudgeableSubmissionStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "partial 可重判", status: SubmissionPartial, want: true},
		{name: "runtime_error 可重判", status: SubmissionRuntimeError, want: true},
		{name: "timeout 可重判", status: SubmissionTimeout, want: true},
		{name: "accepted 不可重判", status: SubmissionAccepted, want: false},
		{name: "pending 不可重判", status: SubmissionPending, want: false},
		{name: "judging 不可重判", status: SubmissionJudging, want: false},
		{name: "空状态不可重判", status: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RejudgeableSubmissionStatus(tt.status); got != tt.want {
				t.Errorf("RejudgeableSubmissionStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
