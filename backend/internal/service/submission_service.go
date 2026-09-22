package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/blueship581/codelearn/internal/constants"
	"github.com/blueship581/codelearn/internal/dto"
	"github.com/blueship581/codelearn/internal/model"
	"github.com/blueship581/codelearn/internal/repository"
	"github.com/blueship581/codelearn/internal/util"
	"github.com/blueship581/codelearn/pkg/strutil"
)

// SubmissionService 提交评测业务：提交 -> 评测 -> 统计与成就联动。
type SubmissionService struct {
	subRepo      *repository.SubmissionRepository
	problemRepo  *repository.ProblemRepository
	userRepo     *repository.UserRepository
	statRepo     *repository.UserStatRepository
	judge        *JudgeService
	achievement  *AchievementService
	logger       *slog.Logger
}

// NewSubmissionService 构造提交评测服务。
func NewSubmissionService(subRepo *repository.SubmissionRepository, problemRepo *repository.ProblemRepository,
	userRepo *repository.UserRepository, statRepo *repository.UserStatRepository, judge *JudgeService,
	achievement *AchievementService, logger *slog.Logger) *SubmissionService {
	return &SubmissionService{
		subRepo:     subRepo,
		problemRepo: problemRepo,
		userRepo:    userRepo,
		statRepo:    statRepo,
		judge:       judge,
		achievement: achievement,
		logger:      logger,
	}
}

// Submit 提交代码并同步评测：ACM 风格逐用例运行。
func (s *SubmissionService) Submit(ctx context.Context, userID primitive.ObjectID, problemID primitive.ObjectID, req *dto.SubmitRequest) (*dto.SubmissionResponse, error) {
	problem, err := s.problemRepo.FindByID(ctx, problemID)
	if err != nil {
		if errors.Is(err, repository.ErrProblemNotFound) {
			return nil, util.WrapAppError(constants.CodeProblemNotFound, constants.MsgProblemNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	if problem.Status != constants.StatusPublished {
		return nil, util.WrapAppError(constants.CodeProblemLocked, constants.MsgProblemLocked, nil)
	}
	if !strutil.Contains(problem.Languages, req.Language) {
		return nil, util.WrapAppError(constants.CodeJudgeLanguage, constants.MsgJudgeLanguage, nil)
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, util.WrapAppError(constants.CodeUserNotFound, constants.MsgUserNotFound, err)
	}

	sub := &model.Submission{
		UserID:       userID,
		Username:     user.Username,
		ProblemID:    problemID,
		ProblemTitle: problem.Title,
		Language:     req.Language,
		Code:         req.Code,
		Status:       constants.SubmissionJudging,
	}
	if err := s.subRepo.Create(ctx, sub); err != nil {
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	s.logger.Info(constants.LogSubmissionCreated, "submission_id", sub.ID.Hex(), "problem_id", problemID.Hex(), "language", req.Language)

	results, status, score, runtimeMs, errMsg := s.judge.Judge(ctx, req.Language, req.Code, problem.TestCases, problem.TimeLimit)
	pointsAwarded := int64(0)
	if status == constants.SubmissionAccepted {
		pointsAwarded = int64(problem.Points)
	}
	if err := s.subRepo.UpdateResult(ctx, sub.ID, status, score, pointsAwarded, runtimeMs, results, errMsg); err != nil {
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}

	// 统计联动（原子操作，无事务依赖）：题目提交数、用户统计。
	_ = s.problemRepo.IncSubmit(ctx, problemID)
	dayKey := util.SignInDailyKey(time.Now())
	_ = s.statRepo.AddSubmission(ctx, userID, req.Language, status == constants.SubmissionAccepted, dayKey)

	if status == constants.SubmissionAccepted {
		// 首次解决才累计积分/解题数/通过数，避免重复刷分；
		// 重判已通过（latest_status=accepted）的提交同样计入去重。
		already, err := s.subRepo.CountSolvedByUser(ctx, userID, problemID)
		if err == nil && already <= 1 {
			_ = s.userRepo.AddPoints(ctx, userID, pointsAwarded)
			_ = s.userRepo.MarkSolved(ctx, userID)
			_ = s.problemRepo.IncAccepted(ctx, problemID)
		}
		// 成就检查：首次通过/完成 N 题/首次通过困难题。
		s.achievement.CheckAfterSubmission(ctx, userID, status, problem)
	}

	sub.Status = status
	sub.Score = score
	sub.PointsAwarded = pointsAwarded
	sub.RuntimeMs = runtimeMs
	sub.Results = results
	sub.ErrorMessage = errMsg
	resp := dto.ToSubmissionResponse(sub)
	return &resp, nil
}

// Get 查询提交记录（本人或管理员）。
func (s *SubmissionService) Get(ctx context.Context, submissionID, userID primitive.ObjectID, role string) (*dto.SubmissionResponse, error) {
	sub, err := s.subRepo.FindByID(ctx, submissionID)
	if err != nil {
		if errors.Is(err, repository.ErrSubmissionNotFound) {
			return nil, util.WrapAppError(constants.CodeSubmissionNotFound, constants.MsgSubmissionNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	if sub.UserID != userID && role != constants.RoleAdmin {
		return nil, util.WrapAppError(constants.CodeSubmissionDenied, constants.MsgSubmissionDenied, nil)
	}
	resp := dto.ToSubmissionResponse(sub)
	return &resp, nil
}

// Rejudge 管理员对"已完成但未通过"的提交发起重判：
// 每次重判生成独立记录（追加到 rejudges），原代码与首次评测结果不改写；
// 重判转为通过时只补发一次积分/解题数/通过数，重复重判或始终失败不扣减、不重复累计。
func (s *SubmissionService) Rejudge(ctx context.Context, submissionID, operatorID primitive.ObjectID) (*dto.SubmissionResponse, error) {
	sub, err := s.subRepo.FindByID(ctx, submissionID)
	if err != nil {
		if errors.Is(err, repository.ErrSubmissionNotFound) {
			return nil, util.WrapAppError(constants.CodeSubmissionNotFound, constants.MsgSubmissionNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	// 仅已完成且未通过（partial/runtime_error/timeout）的最新状态可重判；
	// 已通过（含此前重判通过）或评测中的提交拒绝重判，防止重复补发。
	latestStatus, latestScore := sub.Latest()
	if !constants.RejudgeableSubmissionStatus(latestStatus) {
		s.logger.Warn(constants.LogSubmissionRejudgeDenied, "submission_id", submissionID.Hex(), "latest_status", latestStatus)
		return nil, util.WrapAppError(constants.CodeSubmissionRejudgeDenied, constants.MsgSubmissionRejudgeDenied, nil)
	}
	problem, err := s.problemRepo.FindByID(ctx, sub.ProblemID)
	if err != nil {
		if errors.Is(err, repository.ErrProblemNotFound) {
			return nil, util.WrapAppError(constants.CodeProblemNotFound, constants.MsgProblemNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	operator, err := s.userRepo.FindByID(ctx, operatorID)
	if err != nil {
		return nil, util.WrapAppError(constants.CodeUserNotFound, constants.MsgUserNotFound, err)
	}

	// 使用提交时的原始代码与语言，按题目当前测试用例重新评测。
	results, status, score, runtimeMs, errMsg := s.judge.Judge(ctx, sub.Language, sub.Code, problem.TestCases, problem.TimeLimit)

	// 重判通过且此前未解决该题时，才补发一次积分（幂等去重）。
	pointsAwarded := int64(0)
	if status == constants.SubmissionAccepted {
		solved, cntErr := s.subRepo.CountSolvedByUser(ctx, sub.UserID, sub.ProblemID)
		if cntErr == nil && solved == 0 {
			pointsAwarded = int64(problem.Points)
		}
	}
	rec := &model.RejudgeRecord{
		Seq:           sub.RejudgeCount + 1,
		OperatorID:    operatorID,
		OperatorName:  operator.Username,
		PrevStatus:    latestStatus,
		PrevScore:     latestScore,
		Status:        status,
		Score:         score,
		PointsAwarded: pointsAwarded,
		RuntimeMs:     runtimeMs,
		Results:       results,
		ErrorMessage:  errMsg,
	}
	if err := s.subRepo.AppendRejudge(ctx, sub.ID, rec); err != nil {
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	s.logger.Info(constants.LogSubmissionRejudged,
		"submission_id", sub.ID.Hex(), "seq", rec.Seq, "operator", operator.Username,
		"diff", util.FormatRejudgeDiffText(latestStatus, status, score-latestScore))

	if pointsAwarded > 0 {
		// 补发积分/解题数/通过数（仅一次，由上面的 CountSolvedByUser 去重保证）。
		_ = s.userRepo.AddPoints(ctx, sub.UserID, pointsAwarded)
		_ = s.userRepo.MarkSolved(ctx, sub.UserID)
		_ = s.problemRepo.IncAccepted(ctx, sub.ProblemID)
		// 成就检查：与正常通过一致（授予幂等）。
		s.achievement.CheckAfterSubmission(ctx, sub.UserID, status, problem)
	}

	updated, err := s.subRepo.FindByID(ctx, submissionID)
	if err != nil {
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	resp := dto.ToSubmissionResponse(updated)
	return &resp, nil
}

// List 分页查询提交（支持按用户/题目/状态过滤）。
func (s *SubmissionService) List(ctx context.Context, userID primitive.ObjectID, role string, page, pageSize int64, problemID string, status string) ([]dto.SubmissionResponse, int64, error) {
	filter := bson.M{}
	if problemID != "" {
		pid, err := primitive.ObjectIDFromHex(problemID)
		if err != nil {
			return nil, 0, util.WrapAppError(constants.CodeBadRequest, constants.MsgBadRequest, err)
		}
		filter["problem_id"] = pid
	}
	if status != "" {
		filter["status"] = status
	}
	// 学生只能看自己的提交；管理员可查看全部（按 problem_id/status 过滤）。
	if role != constants.RoleAdmin {
		filter["user_id"] = userID
	}
	subs, total, err := s.subRepo.List(ctx, filter, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	out := make([]dto.SubmissionResponse, 0, len(subs))
	for _, sub := range subs {
		out = append(out, dto.ToSubmissionResponse(sub))
	}
	return out, total, nil
}
