package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/blueship581/codelearn/internal/model"
)

// 提交仓储哨兵错误。
var ErrSubmissionNotFound = errors.New("submission not found")

// SubmissionRepository 提交评测仓储。
type SubmissionRepository struct {
	coll *mongo.Collection
}

// NewSubmissionRepository 构造提交仓储。
func NewSubmissionRepository(db *mongo.Database) *SubmissionRepository {
	return &SubmissionRepository{coll: db.Collection("submissions")}
}

// EnsureIndexes 创建常用查询索引（user_id、problem_id）。
func (r *SubmissionRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "problem_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	})
	return err
}

// Create 创建提交记录（初始 pending）。
func (r *SubmissionRepository) Create(ctx context.Context, s *model.Submission) error {
	s.ID = primitive.NewObjectID()
	s.CreatedAt = time.Now()
	// 重判历史初始化为空数组，避免 nil 序列化为 null 导致后续 $push 失败。
	if s.Rejudges == nil {
		s.Rejudges = []model.RejudgeRecord{}
	}
	_, err := r.coll.InsertOne(ctx, s)
	if err != nil {
		return fmt.Errorf("create submission: %w", err)
	}
	return nil
}

// FindByID 按 ID 查找提交。
func (r *SubmissionRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Submission, error) {
	var s model.Submission
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("find submission by id: %w", ErrSubmissionNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("find submission by id: %w", err)
	}
	return &s, nil
}

// UpdateResult 更新评测结果（同步 latest_status/latest_score 作为最新状态）。
func (r *SubmissionRepository) UpdateResult(ctx context.Context, id primitive.ObjectID, status string, score int, pointsAwarded, runtimeMs int64, results []model.JudgeResult, errMsg string) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"status":         status,
		"score":          score,
		"points_awarded": pointsAwarded,
		"runtime_ms":     runtimeMs,
		"results":        results,
		"error_message":  errMsg,
		"latest_status":  status,
		"latest_score":   score,
	}})
	if err != nil {
		return fmt.Errorf("update submission result: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("update submission result: %w", ErrSubmissionNotFound)
	}
	return nil
}

// AppendRejudge 追加一次重判记录：$push 历史、$inc 重判次数与补发积分、$set 最新状态。
// 首次评测结果（status/score/results 等）不在此处修改。
func (r *SubmissionRepository) AppendRejudge(ctx context.Context, id primitive.ObjectID, rec *model.RejudgeRecord) error {
	rec.ID = primitive.NewObjectID()
	rec.CreatedAt = time.Now()
	// 兼容历史数据：rejudges 为 null 时先归一化为空数组（$push 无法作用于 null）。
	if _, err := r.coll.UpdateOne(ctx, bson.M{"_id": id, "rejudges": nil}, bson.M{"$set": bson.M{"rejudges": []model.RejudgeRecord{}}}); err != nil {
		return fmt.Errorf("normalize submission rejudges: %w", err)
	}
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$push": bson.M{"rejudges": rec},
		"$inc": bson.M{
			"rejudge_count":          1,
			"rejudge_points_awarded": rec.PointsAwarded,
		},
		"$set": bson.M{
			"latest_status": rec.Status,
			"latest_score":  rec.Score,
		},
	})
	if err != nil {
		return fmt.Errorf("append submission rejudge: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("append submission rejudge: %w", ErrSubmissionNotFound)
	}
	return nil
}

// List 分页查询提交（用户/题目/状态过滤）。
func (r *SubmissionRepository) List(ctx context.Context, filter bson.M, skip, limit int64) ([]*model.Submission, int64, error) {
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count submissions: %w", err)
	}
	cur, err := r.coll.Find(ctx, filter, options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, 0, fmt.Errorf("list submissions: %w", err)
	}
	defer cur.Close(ctx)
	subs := make([]*model.Submission, 0)
	for cur.Next(ctx) {
		var s model.Submission
		if err := cur.Decode(&s); err != nil {
			return nil, 0, fmt.Errorf("decode submission: %w", err)
		}
		subs = append(subs, &s)
	}
	return subs, total, nil
}

// AggregatePoints 聚合得分：按日期范围、状态统计每个用户的积分与解题数（排行榜复用）。
// 积分 = 首次评测积分 points_awarded + 重判补发积分 rejudge_points_awarded（缺省按 0 计）。
func (r *SubmissionRepository) AggregatePoints(ctx context.Context, filter bson.M) ([]bson.M, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id":     "$user_id",
			"points":  bson.M{"$sum": bson.M{"$add": bson.A{"$points_awarded", bson.M{"$ifNull": bson.A{"$rejudge_points_awarded", 0}}}}},
			"solved":  bson.M{"$sum": 1},
			"nickname": bson.M{"$last": "$username"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "points", Value: -1}, {Key: "solved", Value: -1}}}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate submission points: %w", err)
	}
	defer cur.Close(ctx)
	out := make([]bson.M, 0)
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode aggregate: %w", err)
		}
		out = append(out, doc)
	}
	return out, nil
}

// CountSolvedByUser 统计用户在某题目"已解决"的提交数：
// 首次评测通过（status=accepted）或重判后通过（latest_status=accepted）均计为已解决。
// 提交去重与重判补发积分的幂等判定复用此方法，防止重复累计。
func (r *SubmissionRepository) CountSolvedByUser(ctx context.Context, userID, problemID primitive.ObjectID) (int64, error) {
	n, err := r.coll.CountDocuments(ctx, bson.M{
		"user_id":    userID,
		"problem_id": problemID,
		"$or": bson.A{
			bson.M{"status": "accepted"},
			bson.M{"latest_status": "accepted"},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("count solved by user: %w", err)
	}
	return n, nil
}
