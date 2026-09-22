package dto

import (
	"github.com/blueship581/codelearn/internal/model"
)

// RejudgeDiffResponse 单次重判相对上一轮的结果差异。
type RejudgeDiffResponse struct {
	StatusChanged  bool   `json:"status_changed"`
	FromStatus     string `json:"from_status"`
	ToStatus       string `json:"to_status"`
	ScoreDelta     int    `json:"score_delta"`
	RuntimeDeltaMs int64  `json:"runtime_delta_ms"`
	ChangedCases   []int  `json:"changed_cases"`
}

// RejudgeResponse 重判记录响应（每次重判独立、不可变）。
type RejudgeResponse struct {
	ID             string                `json:"id"`
	SubmissionID   string                `json:"submission_id"`
	Round          int                   `json:"round"`
	Status         string                `json:"status"`
	Score          int                   `json:"score"`
	PointsAwarded  int64                 `json:"points_awarded"`
	RuntimeMs      int64                 `json:"runtime_ms"`
	Results        []JudgeResultResponse `json:"results"`
	ErrorMessage   string                `json:"error_message"`
	RewardsGranted bool                  `json:"rewards_granted"`
	Diff           RejudgeDiffResponse   `json:"diff"`
	OperatorName   string                `json:"operator_name"`
	CreatedAt      string                `json:"created_at"`
}

// ToRejudgeResponse 将重判模型转换为响应。
func ToRejudgeResponse(r *model.Rejudge) RejudgeResponse {
	results := make([]JudgeResultResponse, 0, len(r.Results))
	for _, res := range r.Results {
		results = append(results, JudgeResultResponse{
			TestCaseIndex: res.TestCaseIndex,
			Input:         res.Input,
			Expected:      res.Expected,
			Actual:        res.Actual,
			Passed:        res.Passed,
			ErrorMessage:  res.ErrorMessage,
		})
	}
	changed := r.Diff.ChangedCases
	if changed == nil {
		changed = []int{}
	}
	return RejudgeResponse{
		ID:             r.ID.Hex(),
		SubmissionID:   r.SubmissionID.Hex(),
		Round:          r.Round,
		Status:         r.Status,
		Score:          r.Score,
		PointsAwarded:  r.PointsAwarded,
		RuntimeMs:      r.RuntimeMs,
		Results:        results,
		ErrorMessage:   r.ErrorMessage,
		RewardsGranted: r.RewardsGranted,
		Diff: RejudgeDiffResponse{
			StatusChanged:  r.Diff.StatusChanged,
			FromStatus:     r.Diff.FromStatus,
			ToStatus:       r.Diff.ToStatus,
			ScoreDelta:     r.Diff.ScoreDelta,
			RuntimeDeltaMs: r.Diff.RuntimeDeltaMs,
			ChangedCases:   changed,
		},
		OperatorName: r.OperatorName,
		CreatedAt:    r.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
