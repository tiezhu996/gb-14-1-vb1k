package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 重判来源：original 表示首次评测（基线，不单独入库），
// rejudge 表示管理员发起的重判。
const (
	JudgeSourceOriginal = "original"
	JudgeSourceRejudge  = "rejudge"
)

// RejudgeDiff 本轮评测相对上一轮的差异（原代码不参与比对，仅比对评测结果）。
type RejudgeDiff struct {
	StatusChanged  bool   `bson:"status_changed" json:"status_changed"`
	FromStatus     string `bson:"from_status" json:"from_status"`
	ToStatus       string `bson:"to_status" json:"to_status"`
	ScoreDelta     int    `bson:"score_delta" json:"score_delta"`
	RuntimeDeltaMs int64  `bson:"runtime_delta_ms" json:"runtime_delta_ms"`
	// ChangedCases 状态发生翻转（通过<->未通过）的用例下标。
	ChangedCases []int `bson:"changed_cases" json:"changed_cases"`
}

// Rejudge 提交重判记录：每次管理员发起重判都生成一条独立、不可变的记录。
// 原提交的代码（submissions.code）与首次评测结果（submissions.results 等）不会被改写。
type Rejudge struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SubmissionID  primitive.ObjectID `bson:"submission_id" json:"submission_id"`
	ProblemID     primitive.ObjectID `bson:"problem_id" json:"problem_id"`
	UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
	Username      string             `bson:"username" json:"username"`
	Round         int                `bson:"round" json:"round"`
	Status        string             `bson:"status" json:"status"`
	Score         int                `bson:"score" json:"score"`
	PointsAwarded int64              `bson:"points_awarded" json:"points_awarded"`
	RuntimeMs     int64              `bson:"runtime_ms" json:"runtime_ms"`
	Results       []JudgeResult      `bson:"results" json:"results"`
	ErrorMessage  string             `bson:"error_message" json:"error_message"`
	// RewardsGranted 本次重判转为通过时是否实际补发了积分/解决数/通过数（幂等只补一次）。
	RewardsGranted bool        `bson:"rewards_granted" json:"rewards_granted"`
	Diff           RejudgeDiff `bson:"diff" json:"diff"`
	// OperatorID/OperatorName 发起重判的管理员（审计留痕）。
	OperatorID   primitive.ObjectID `bson:"operator_id" json:"operator_id"`
	OperatorName string             `bson:"operator_name" json:"operator_name"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}
