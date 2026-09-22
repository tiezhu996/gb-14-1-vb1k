package service

import (
	"testing"

	"github.com/blueship581/codelearn/internal/constants"
	"github.com/blueship581/codelearn/internal/model"
)

func TestBuildRejudgeDiff_StillFailing(t *testing.T) {
	prev := []model.JudgeResult{
		{TestCaseIndex: 0, Passed: false},
		{TestCaseIndex: 1, Passed: false},
	}
	cur := []model.JudgeResult{
		{TestCaseIndex: 0, Passed: true},
		{TestCaseIndex: 1, Passed: false},
	}
	diff := buildRejudgeDiff(constants.SubmissionRuntimeError, 0, 100, prev,
		constants.SubmissionPartial, 50, 120, cur)
	if diff.StatusChanged != true {
		t.Errorf("status should change runtime_error -> partial")
	}
	if diff.ScoreDelta != 50 {
		t.Errorf("score delta = %d, want 50", diff.ScoreDelta)
	}
	if diff.RuntimeDeltaMs != 20 {
		t.Errorf("runtime delta = %d, want 20", diff.RuntimeDeltaMs)
	}
	if len(diff.ChangedCases) != 1 || diff.ChangedCases[0] != 0 {
		t.Errorf("changed cases = %v, want [0]", diff.ChangedCases)
	}
}

func TestBuildRejudgeDiff_TurnAccepted(t *testing.T) {
	prev := []model.JudgeResult{
		{TestCaseIndex: 0, Passed: false},
		{TestCaseIndex: 1, Passed: true},
	}
	cur := []model.JudgeResult{
		{TestCaseIndex: 0, Passed: true},
		{TestCaseIndex: 1, Passed: true},
	}
	diff := buildRejudgeDiff(constants.SubmissionPartial, 50, 10, prev,
		constants.SubmissionAccepted, 100, 8, cur)
	if !diff.StatusChanged || diff.FromStatus != constants.SubmissionPartial || diff.ToStatus != constants.SubmissionAccepted {
		t.Errorf("unexpected status change: %+v", diff)
	}
	if diff.ScoreDelta != 50 {
		t.Errorf("score delta = %d, want 50", diff.ScoreDelta)
	}
	if len(diff.ChangedCases) != 1 || diff.ChangedCases[0] != 0 {
		t.Errorf("changed cases = %v, want [0]", diff.ChangedCases)
	}
}

func TestBuildRejudgeDiff_NoChange(t *testing.T) {
	prev := []model.JudgeResult{{TestCaseIndex: 0, Passed: false}}
	cur := []model.JudgeResult{{TestCaseIndex: 0, Passed: false}}
	diff := buildRejudgeDiff(constants.SubmissionTimeout, 0, 5, prev,
		constants.SubmissionTimeout, 0, 6, cur)
	if diff.StatusChanged {
		t.Errorf("status should not change")
	}
	if diff.ScoreDelta != 0 {
		t.Errorf("score delta = %d, want 0", diff.ScoreDelta)
	}
	if len(diff.ChangedCases) != 0 {
		t.Errorf("changed cases = %v, want empty", diff.ChangedCases)
	}
}

func TestRejudgeEligibleStatus(t *testing.T) {
	allowed := []string{constants.SubmissionPartial, constants.SubmissionRuntimeError, constants.SubmissionTimeout}
	for _, st := range allowed {
		if !constants.RejudgeEligibleStatus(st) {
			t.Errorf("status %s should be rejudge-eligible", st)
		}
	}
	denied := []string{constants.SubmissionAccepted, constants.SubmissionPending, constants.SubmissionJudging, ""}
	for _, st := range denied {
		if constants.RejudgeEligibleStatus(st) {
			t.Errorf("status %q should NOT be rejudge-eligible", st)
		}
	}
}

func TestSubmissionCurrentStatus(t *testing.T) {
	// 历史文档无 latest_status，回退首次状态。
	old := &model.Submission{Status: constants.SubmissionRuntimeError}
	if got := old.CurrentStatus(); got != constants.SubmissionRuntimeError {
		t.Errorf("old doc current status = %s, want runtime_error", got)
	}
	// 重判后以最新状态为准。
	rejudged := &model.Submission{Status: constants.SubmissionRuntimeError, LatestStatus: constants.SubmissionAccepted}
	if got := rejudged.CurrentStatus(); got != constants.SubmissionAccepted {
		t.Errorf("rejudged current status = %s, want accepted", got)
	}
}
