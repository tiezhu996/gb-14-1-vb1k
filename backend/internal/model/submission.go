package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JudgeResult 单个测试用例的评测结果。
type JudgeResult struct {
	TestCaseIndex int    `bson:"test_case_index" json:"test_case_index"`
	Input         string `bson:"input" json:"input"`
	Expected      string `bson:"expected" json:"expected"`
	Actual        string `bson:"actual" json:"actual"`
	Passed        bool   `bson:"passed" json:"passed"`
	ErrorMessage  string `bson:"error_message" json:"error_message"`
}

// RejudgeRecord 一次重判的独立记录：追加写入、永不改写。
// PrevStatus/PrevScore 记录重判前的最新结果，用于展示历次结果差异。
type RejudgeRecord struct {
	ID            primitive.ObjectID `bson:"_id" json:"id"`
	Seq           int                `bson:"seq" json:"seq"`
	OperatorID    primitive.ObjectID `bson:"operator_id" json:"operator_id"`
	OperatorName  string             `bson:"operator_name" json:"operator_name"`
	PrevStatus    string             `bson:"prev_status" json:"prev_status"`
	PrevScore     int                `bson:"prev_score" json:"prev_score"`
	Status        string             `bson:"status" json:"status"`
	Score         int                `bson:"score" json:"score"`
	PointsAwarded int64              `bson:"points_awarded" json:"points_awarded"`
	RuntimeMs     int64              `bson:"runtime_ms" json:"runtime_ms"`
	Results       []JudgeResult      `bson:"results" json:"results"`
	ErrorMessage  string             `bson:"error_message" json:"error_message"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
}

// Submission 提交评测实体：状态机 pending -> judging -> accepted/partial/runtime_error/timeout。
// Status/Score/Results 等字段保存首次评测结果，重判不改写；
// 重判结果追加到 Rejudges，并同步 LatestStatus/LatestScore/RejudgeCount。
type Submission struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"user_id" json:"user_id"`
	Username     string             `bson:"username" json:"username"`
	ProblemID    primitive.ObjectID `bson:"problem_id" json:"problem_id"`
	ProblemTitle string             `bson:"problem_title" json:"problem_title"`
	Language     string             `bson:"language" json:"language"`
	Code         string             `bson:"code" json:"code"`
	Status       string             `bson:"status" json:"status"`
	Score        int                `bson:"score" json:"score"`
	PointsAwarded int64             `bson:"points_awarded" json:"points_awarded"`
	RuntimeMs    int64              `bson:"runtime_ms" json:"runtime_ms"`
	Results      []JudgeResult      `bson:"results" json:"results"`
	ErrorMessage string             `bson:"error_message" json:"error_message"`
	// 重判扩展字段：最新状态/得分（无重判时等于首次结果）、重判次数与历史。
	LatestStatus         string         `bson:"latest_status" json:"latest_status"`
	LatestScore          int            `bson:"latest_score" json:"latest_score"`
	RejudgeCount         int            `bson:"rejudge_count" json:"rejudge_count"`
	RejudgePointsAwarded int64          `bson:"rejudge_points_awarded" json:"rejudge_points_awarded"`
	Rejudges             []RejudgeRecord `bson:"rejudges" json:"rejudges"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

// Latest 返回最新状态与得分（兼容无重判扩展字段的历史数据）。
func (s *Submission) Latest() (string, int) {
	status := s.LatestStatus
	if status == "" {
		status = s.Status
	}
	score := s.LatestScore
	if s.LatestStatus == "" {
		score = s.Score
	}
	return status, score
}
