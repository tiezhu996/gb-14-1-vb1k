package model

import "testing"

func TestSubmissionLatestFallback(t *testing.T) {
	// 无重判扩展字段的历史数据：最新状态回退为首次评测结果。
	legacy := &Submission{Status: "runtime_error", Score: 0}
	status, score := legacy.Latest()
	if status != "runtime_error" || score != 0 {
		t.Errorf("Latest() = (%q, %d), want (runtime_error, 0)", status, score)
	}

	// 有重判记录：最新状态以 latest_status/latest_score 为准，首次结果不改写。
	rejudged := &Submission{Status: "partial", Score: 50, LatestStatus: "accepted", LatestScore: 100, RejudgeCount: 1}
	status, score = rejudged.Latest()
	if status != "accepted" || score != 100 {
		t.Errorf("Latest() = (%q, %d), want (accepted, 100)", status, score)
	}
	if rejudged.Status != "partial" || rejudged.Score != 50 {
		t.Errorf("first result rewritten: status=%q score=%d", rejudged.Status, rejudged.Score)
	}
}
