package dto

import (
	"github.com/blueship581/codelearn/internal/model"
)

// SubmitRequest 提交代码请求。
type SubmitRequest struct {
	Language string `json:"language" binding:"required,oneof=python javascript java"`
	Code     string `json:"code" binding:"required,min=1,max=20000"`
}

// JudgeResultResponse 单个测试用例评测结果。
type JudgeResultResponse struct {
	TestCaseIndex int    `json:"test_case_index"`
	Input         string `json:"input"`
	Expected      string `json:"expected"`
	Actual        string `json:"actual"`
	Passed        bool   `json:"passed"`
	ErrorMessage  string `json:"error_message"`
}

// RejudgeRecordResponse 单次重判记录（含与上一次结果的差异）。
type RejudgeRecordResponse struct {
	ID            string                `json:"id"`
	Seq           int                   `json:"seq"`
	OperatorID    string                `json:"operator_id"`
	OperatorName  string                `json:"operator_name"`
	PrevStatus    string                `json:"prev_status"`
	PrevScore     int                   `json:"prev_score"`
	Status        string                `json:"status"`
	Score         int                   `json:"score"`
	ScoreDelta    int                   `json:"score_delta"`
	PointsAwarded int64                 `json:"points_awarded"`
	RuntimeMs     int64                 `json:"runtime_ms"`
	Results       []JudgeResultResponse `json:"results"`
	ErrorMessage  string                `json:"error_message"`
	CreatedAt     string                `json:"created_at"`
}

// SubmissionResponse 提交记录响应。
type SubmissionResponse struct {
	ID            string                `json:"id"`
	UserID        string                `json:"user_id"`
	Username      string                `json:"username"`
	ProblemID     string                `json:"problem_id"`
	ProblemTitle  string                `json:"problem_title"`
	Language      string                `json:"language"`
	Code          string                `json:"code"`
	Status        string                `json:"status"`
	Score         int                   `json:"score"`
	PointsAwarded int64                 `json:"points_awarded"`
	RuntimeMs     int64                 `json:"runtime_ms"`
	Results       []JudgeResultResponse `json:"results"`
	ErrorMessage  string                `json:"error_message"`
	// 重判扩展：最新状态/得分、重判次数与历次重判记录。
	LatestStatus         string                  `json:"latest_status"`
	LatestScore          int                     `json:"latest_score"`
	RejudgeCount         int                     `json:"rejudge_count"`
	RejudgePointsAwarded int64                   `json:"rejudge_points_awarded"`
	Rejudges             []RejudgeRecordResponse `json:"rejudges"`
	CreatedAt            string                  `json:"created_at"`
}

// ToSubmissionResponse 将模型转换为响应。
func ToSubmissionResponse(s *model.Submission) SubmissionResponse {
	results := make([]JudgeResultResponse, 0, len(s.Results))
	for _, r := range s.Results {
		results = append(results, JudgeResultResponse{
			TestCaseIndex: r.TestCaseIndex,
			Input:         r.Input,
			Expected:      r.Expected,
			Actual:        r.Actual,
			Passed:        r.Passed,
			ErrorMessage:  r.ErrorMessage,
		})
	}
	latestStatus, latestScore := s.Latest()
	rejudges := make([]RejudgeRecordResponse, 0, len(s.Rejudges))
	for _, r := range s.Rejudges {
		rejudges = append(rejudges, ToRejudgeRecordResponse(&r))
	}
	return SubmissionResponse{
		ID:                   s.ID.Hex(),
		UserID:               s.UserID.Hex(),
		Username:             s.Username,
		ProblemID:            s.ProblemID.Hex(),
		ProblemTitle:         s.ProblemTitle,
		Language:             s.Language,
		Code:                 s.Code,
		Status:               s.Status,
		Score:                s.Score,
		PointsAwarded:        s.PointsAwarded,
		RuntimeMs:            s.RuntimeMs,
		Results:              results,
		ErrorMessage:         s.ErrorMessage,
		LatestStatus:         latestStatus,
		LatestScore:          latestScore,
		RejudgeCount:         s.RejudgeCount,
		RejudgePointsAwarded: s.RejudgePointsAwarded,
		Rejudges:             rejudges,
		CreatedAt:            s.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ToRejudgeRecordResponse 将重判记录转换为响应（score_delta 为与上一次结果的得分差）。
func ToRejudgeRecordResponse(r *model.RejudgeRecord) RejudgeRecordResponse {
	results := make([]JudgeResultResponse, 0, len(r.Results))
	for _, jr := range r.Results {
		results = append(results, JudgeResultResponse{
			TestCaseIndex: jr.TestCaseIndex,
			Input:         jr.Input,
			Expected:      jr.Expected,
			Actual:        jr.Actual,
			Passed:        jr.Passed,
			ErrorMessage:  jr.ErrorMessage,
		})
	}
	return RejudgeRecordResponse{
		ID:            r.ID.Hex(),
		Seq:           r.Seq,
		OperatorID:    r.OperatorID.Hex(),
		OperatorName:  r.OperatorName,
		PrevStatus:    r.PrevStatus,
		PrevScore:     r.PrevScore,
		Status:        r.Status,
		Score:         r.Score,
		ScoreDelta:    r.Score - r.PrevScore,
		PointsAwarded: r.PointsAwarded,
		RuntimeMs:     r.RuntimeMs,
		Results:       results,
		ErrorMessage:  r.ErrorMessage,
		CreatedAt:     r.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
