package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/blueship581/codelearn/internal/constants"
	"github.com/blueship581/codelearn/internal/dto"
	"github.com/blueship581/codelearn/internal/model"
	"github.com/blueship581/codelearn/internal/repository"
	"github.com/blueship581/codelearn/internal/util"
)

// RejudgeService 提交重判业务。
//
// 不变量（需求约束）：
//  1. 仅管理员可对“已完成但未通过”（partial/runtime_error/timeout）的提交发起重判；
//  2. 每次重判生成独立不可变记录（submission_rejudges），原代码与首次评测结果永不改写；
//  3. 重判转为通过时只补发一次积分、解决数、通过数（submissions.rewards_granted 原子幂等）；
//  4. 始终失败或重复重判不扣减、不重复累计；
//  5. 普通提交与首次评测流程（SubmissionService.Submit）不受影响。
type RejudgeService struct {
	rejudgeRepo *repository.RejudgeRepository
	subRepo     *repository.SubmissionRepository
	problemRepo *repository.ProblemRepository
	userRepo    *repository.UserRepository
	statRepo    *repository.UserStatRepository
	judge       *JudgeService
	logger      *slog.Logger
}

// NewRejudgeService 构造重判服务。
func NewRejudgeService(rejudgeRepo *repository.RejudgeRepository, subRepo *repository.SubmissionRepository,
	problemRepo *repository.ProblemRepository, userRepo *repository.UserRepository,
	statRepo *repository.UserStatRepository, judge *JudgeService, logger *slog.Logger) *RejudgeService {
	return &RejudgeService{
		rejudgeRepo: rejudgeRepo,
		subRepo:     subRepo,
		problemRepo: problemRepo,
		userRepo:    userRepo,
		statRepo:    statRepo,
		judge:       judge,
		logger:      logger,
	}
}

// Rejudge 管理员对一条提交发起重判，返回新建的独立重判记录。
func (s *RejudgeService) Rejudge(ctx context.Context, submissionID, operatorID primitive.ObjectID, operatorName string) (*dto.RejudgeResponse, error) {
	sub, err := s.subRepo.FindByID(ctx, submissionID)
	if err != nil {
		if errors.Is(err, repository.ErrSubmissionNotFound) {
			return nil, util.WrapAppError(constants.CodeSubmissionNotFound, constants.MsgSubmissionNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}

	// 仅“已完成但未通过”可重判：以最新状态为准（重复重判若已通过则禁止再判）。
	current := sub.CurrentStatus()
	if !constants.RejudgeEligibleStatus(current) {
		s.logger.Warn(constants.LogRejudgeDenied, "submission_id", submissionID.Hex(), "current_status", current, "operator", operatorName)
		return nil, util.WrapAppError(constants.CodeRejudgeNotEligible, constants.MsgRejudgeNotEligible, nil)
	}

	problem, err := s.problemRepo.FindByID(ctx, sub.ProblemID)
	if err != nil {
		if errors.Is(err, repository.ErrProblemNotFound) {
			return nil, util.WrapAppError(constants.CodeProblemNotFound, constants.MsgProblemNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	if len(problem.TestCases) == 0 {
		return nil, util.WrapAppError(constants.CodeRejudgeNotEligible, constants.MsgRejudgeNoCases, nil)
	}

	// 原子占用轮次：并发重判各自拿到不同 round，原提交字段不修改。
	round, err := s.subRepo.ReserveRejudgeRound(ctx, submissionID)
	if err != nil {
		if errors.Is(err, repository.ErrSubmissionNotFound) {
			return nil, util.WrapAppError(constants.CodeSubmissionNotFound, constants.MsgSubmissionNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgRejudgeRoundFailed, err)
	}
	s.logger.Info(constants.LogRejudgeRoundReserved, "submission_id", submissionID.Hex(), "round", round, "operator", operatorName)

	// 重判使用当前题目测试用例，但运行的是原提交代码（不可变）。
	// 评测沙箱异常时回滚轮次占用，避免 latest_status 永久停留在 judging。
	results, status, score, runtimeMs, errMsg := func() (results []model.JudgeResult, status string, score int, runtimeMs int64, errMsg string) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("rejudge judge panic", "submission_id", submissionID.Hex(), "round", round, "panic", r)
				_ = s.subRepo.ResetRejudgeForRoundFailed(ctx, submissionID)
				panic(r) // 交由全局 error_handler 统一返回 500
			}
		}()
		return s.judge.Judge(ctx, sub.Language, sub.Code, problem.TestCases, problem.TimeLimit)
	}()
	pointsAwarded := int64(0)
	if status == constants.SubmissionAccepted {
		pointsAwarded = int64(problem.Points)
	}

	// 差异相对“上一轮”：第 1 次重判对比首次评测，其后对比最近一次重判。
	prevStatus, prevScore, prevRuntime, prevResults := s.previousOutcome(ctx, sub, round)
	diff := buildRejudgeDiff(prevStatus, prevScore, prevRuntime, prevResults, status, score, runtimeMs, results)

	rejudged := &model.Rejudge{
		SubmissionID:  sub.ID,
		ProblemID:     sub.ProblemID,
		UserID:        sub.UserID,
		Username:      sub.Username,
		Round:         round,
		Status:        status,
		Score:         score,
		PointsAwarded: pointsAwarded,
		RuntimeMs:     runtimeMs,
		Results:       results,
		ErrorMessage:  errMsg,
		Diff:          diff,
		OperatorID:    operatorID,
		OperatorName:  operatorName,
	}

	// 重判通过：原子抢占补发资格，保证积分/解决数/通过数只补一次。
	rewarded := false
	if status == constants.SubmissionAccepted {
		got, err := s.subRepo.MarkRewarded(ctx, sub.ID)
		if err != nil {
			s.logger.Error("rejudge mark rewarded failed", "submission_id", sub.ID.Hex(), "error", err.Error())
			got = false
		}
		if got {
			// 积分、解决数（用户）与通过数（题目）只在此分支补发一次。
			if err := s.userRepo.AddPoints(ctx, sub.UserID, pointsAwarded); err != nil {
				s.logger.Error("rejudge add points failed", "submission_id", sub.ID.Hex(), "error", err.Error())
			}
			if err := s.userRepo.MarkSolved(ctx, sub.UserID); err != nil {
				s.logger.Error("rejudge mark solved failed", "submission_id", sub.ID.Hex(), "error", err.Error())
			}
			if err := s.problemRepo.IncAccepted(ctx, sub.ProblemID); err != nil {
				s.logger.Error("rejudge inc accepted failed", "problem_id", sub.ProblemID.Hex(), "error", err.Error())
			}
			// 用户通过数（user_stats）只补通过、不补总提交。
			dayKey := util.SignInDailyKey(time.Now())
			if err := s.statRepo.AddAcceptedOnly(ctx, sub.UserID, sub.Language, dayKey); err != nil {
				s.logger.Error("rejudge stat accepted failed", "submission_id", sub.ID.Hex(), "error", err.Error())
			}
			rewarded = true
			s.logger.Info(constants.LogRejudgeRewardsGranted, "submission_id", sub.ID.Hex(), "round", round, "points", pointsAwarded)
		} else {
			// 此前已补发过（重复重判）：不重复累计。
			s.logger.Info(constants.LogRejudgeRewardsSkipped, "submission_id", sub.ID.Hex(), "round", round)
		}
		rejudged.RewardsGranted = rewarded
	}

	// 落独立记录；若因并发产生同 round 记录（唯一索引兜底），回滚轮次。
	if err := s.rejudgeRepo.Create(ctx, rejudged); err != nil {
		_ = s.subRepo.ResetRejudgeForRoundFailed(ctx, sub.ID)
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}

	// 更新最新状态：已通过后不再被后续重判覆盖回失败。
	if err := s.subRepo.SetLatestStatusIfNotAccepted(ctx, sub.ID, status); err != nil {
		s.logger.Error("rejudge set latest status failed", "submission_id", sub.ID.Hex(), "error", err.Error())
	}

	if status == constants.SubmissionAccepted {
		s.logger.Info(constants.LogRejudgeAccepted, "submission_id", sub.ID.Hex(), "round", round, "rewarded", rewarded)
	} else {
		// 始终失败：不扣减任何统计。
		s.logger.Info(constants.LogRejudgeStillFailed, "submission_id", sub.ID.Hex(), "round", round, "status", status)
	}

	resp := dto.ToRejudgeResponse(rejudged)
	return &resp, nil
}

// ListHistory 查询提交的历次重判记录（本人或管理员；学生只能查看本人记录）。
func (s *RejudgeService) ListHistory(ctx context.Context, submissionID, userID primitive.ObjectID, role string) ([]dto.RejudgeResponse, error) {
	if err := s.authorize(ctx, submissionID, userID, role); err != nil {
		return nil, err
	}
	list, err := s.rejudgeRepo.ListBySubmission(ctx, submissionID)
	if err != nil {
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	out := make([]dto.RejudgeResponse, 0, len(list))
	for _, rj := range list {
		out = append(out, dto.ToRejudgeResponse(rj))
	}
	return out, nil
}

// authorize 校验提交存在且访问者为本人或管理员（学生只能查看本人记录）。
func (s *RejudgeService) authorize(ctx context.Context, submissionID, userID primitive.ObjectID, role string) error {
	sub, err := s.subRepo.FindByID(ctx, submissionID)
	if err != nil {
		if errors.Is(err, repository.ErrSubmissionNotFound) {
			return util.WrapAppError(constants.CodeSubmissionNotFound, constants.MsgSubmissionNotFound, err)
		}
		return util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	if sub.UserID != userID && role != constants.RoleAdmin {
		return util.WrapAppError(constants.CodeSubmissionDenied, constants.MsgSubmissionDenied, nil)
	}
	return nil
}

// previousOutcome 取上一轮评测结果：第 1 次重判对比首次评测，否则对比最近一条重判记录。
func (s *RejudgeService) previousOutcome(ctx context.Context, sub *model.Submission, round int) (string, int, int64, []model.JudgeResult) {
	if round <= 1 {
		return sub.Status, sub.Score, sub.RuntimeMs, sub.Results
	}
	history, err := s.rejudgeRepo.ListBySubmission(ctx, sub.ID)
	if err != nil || len(history) == 0 {
		return sub.Status, sub.Score, sub.RuntimeMs, sub.Results
	}
	last := history[len(history)-1]
	return last.Status, last.Score, last.RuntimeMs, last.Results
}

// buildRejudgeDiff 计算本轮相对上一轮的差异（纯函数，便于测试）。
func buildRejudgeDiff(prevStatus string, prevScore int, prevRuntime int64, prevResults []model.JudgeResult,
	status string, score int, runtime int64, results []model.JudgeResult) model.RejudgeDiff {
	diff := model.RejudgeDiff{
		FromStatus:     prevStatus,
		ToStatus:       status,
		StatusChanged:  prevStatus != status,
		ScoreDelta:     score - prevScore,
		RuntimeDeltaMs: runtime - prevRuntime,
		ChangedCases:   []int{},
	}
	prevPassed := make(map[int]bool, len(prevResults))
	for _, r := range prevResults {
		prevPassed[r.TestCaseIndex] = r.Passed
	}
	for _, r := range results {
		// 仅当该用例通过状态发生翻转（通过<->未通过）才计入差异。
		if prev, ok := prevPassed[r.TestCaseIndex]; ok && prev != r.Passed {
			diff.ChangedCases = append(diff.ChangedCases, r.TestCaseIndex)
		}
	}
	return diff
}
