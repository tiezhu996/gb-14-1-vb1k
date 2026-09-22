package dto

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/blueship581/codelearn/internal/model"
)

func TestToSubmissionResponseRejudge(t *testing.T) {
	sub := &model.Submission{
		ID:           primitive.NewObjectID(),
		UserID:       primitive.NewObjectID(),
		ProblemID:    primitive.NewObjectID(),
		Language:     "python",
		Status:       "runtime_error",
		Score:        0,
		LatestStatus: "accepted",
		LatestScore:  100,
		RejudgeCount: 1,
		Rejudges: []model.RejudgeRecord{
			{
				ID:         primitive.NewObjectID(),
				Seq:        1,
				OperatorID: primitive.NewObjectID(),
				PrevStatus: "runtime_error",
				PrevScore:  0,
				Status:     "accepted",
				Score:      100,
				CreatedAt:  time.Now(),
			},
		},
		CreatedAt: time.Now(),
	}
	resp := ToSubmissionResponse(sub)
	// 首次评测结果保持原样，最新状态来自重判。
	if resp.Status != "runtime_error" || resp.Score != 0 {
		t.Errorf("first result changed: status=%q score=%d", resp.Status, resp.Score)
	}
	if resp.LatestStatus != "accepted" || resp.LatestScore != 100 {
		t.Errorf("latest = (%q, %d), want (accepted, 100)", resp.LatestStatus, resp.LatestScore)
	}
	if resp.RejudgeCount != 1 || len(resp.Rejudges) != 1 {
		t.Fatalf("rejudge history missing: count=%d len=%d", resp.RejudgeCount, len(resp.Rejudges))
	}
	if resp.Rejudges[0].ScoreDelta != 100 {
		t.Errorf("score_delta = %d, want 100", resp.Rejudges[0].ScoreDelta)
	}
}

func TestToSubmissionResponseLegacyFallback(t *testing.T) {
	sub := &model.Submission{
		ID:        primitive.NewObjectID(),
		UserID:    primitive.NewObjectID(),
		ProblemID: primitive.NewObjectID(),
		Status:    "partial",
		Score:     50,
		CreatedAt: time.Now(),
	}
	resp := ToSubmissionResponse(sub)
	if resp.LatestStatus != "partial" || resp.LatestScore != 50 {
		t.Errorf("legacy latest = (%q, %d), want (partial, 50)", resp.LatestStatus, resp.LatestScore)
	}
	if resp.RejudgeCount != 0 || len(resp.Rejudges) != 0 {
		t.Errorf("unexpected rejudge history: count=%d len=%d", resp.RejudgeCount, len(resp.Rejudges))
	}
}
