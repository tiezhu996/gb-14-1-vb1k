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

// Submission 提交评测实体：状态机 pending -> judging -> accepted/partial/runtime_error/timeout。
//
// 重判相关说明（原代码与首次评测结果不可变）：
//   - Code/Status/Score/PointsAwarded/RuntimeMs/Results/ErrorMessage 始终保留首次提交评测的值；
//   - RejudgeCount 记录管理员发起的重判次数（每次重判 +1，独立记录见 Rejudge 集合）；
//   - LatestStatus 为最新状态（无重判时与 Status 相同），列表与详情优先展示该字段；
//   - RewardsGranted 标记该提交是否已因“通过”补发过积分/解决数/通过数，保证只补一次。
type Submission struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
	Username      string             `bson:"username" json:"username"`
	ProblemID     primitive.ObjectID `bson:"problem_id" json:"problem_id"`
	ProblemTitle  string             `bson:"problem_title" json:"problem_title"`
	Language      string             `bson:"language" json:"language"`
	Code          string             `bson:"code" json:"code"`
	Status        string             `bson:"status" json:"status"`
	Score         int                `bson:"score" json:"score"`
	PointsAwarded int64              `bson:"points_awarded" json:"points_awarded"`
	RuntimeMs     int64              `bson:"runtime_ms" json:"runtime_ms"`
	Results       []JudgeResult      `bson:"results" json:"results"`
	ErrorMessage  string             `bson:"error_message" json:"error_message"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`

	// 重判扩展字段（普通提交流程不写入，零值兼容历史文档）。
	RejudgeCount   int    `bson:"rejudge_count" json:"rejudge_count"`
	LatestStatus   string `bson:"latest_status,omitempty" json:"latest_status,omitempty"`
	RewardsGranted bool   `bson:"rewards_granted" json:"rewards_granted"`
}

// CurrentStatus 返回提交的最新状态：有重判则取 LatestStatus，否则取首次评测状态。
func (s *Submission) CurrentStatus() string {
	if s.LatestStatus != "" {
		return s.LatestStatus
	}
	return s.Status
}
