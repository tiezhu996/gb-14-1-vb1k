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

// 重判仓储哨兵错误。
var ErrRejudgeNotFound = errors.New("rejudge not found")

// RejudgeRepository 提交重判记录仓储：每次重判独立、不可变，仅追加不改写。
type RejudgeRepository struct {
	coll *mongo.Collection
}

// NewRejudgeRepository 构造重判仓储。
func NewRejudgeRepository(db *mongo.Database) *RejudgeRepository {
	return &RejudgeRepository{coll: db.Collection("submission_rejudges")}
}

// EnsureIndexes 创建查询索引与 (submission_id, round) 唯一索引（并发重判防重号）。
func (r *RejudgeRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "submission_id", Value: 1}, {Key: "round", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "problem_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
	})
	return err
}

// Create 追加一条重判记录（独立记录，不影响首次评测数据）。
func (r *RejudgeRepository) Create(ctx context.Context, rj *model.Rejudge) error {
	rj.ID = primitive.NewObjectID()
	rj.CreatedAt = time.Now()
	_, err := r.coll.InsertOne(ctx, rj)
	if err != nil {
		return fmt.Errorf("create rejudge: %w", err)
	}
	return nil
}

// ListBySubmission 按提交查询历次重判（按轮次升序，便于回读历史与差异）。
func (r *RejudgeRepository) ListBySubmission(ctx context.Context, submissionID primitive.ObjectID) ([]*model.Rejudge, error) {
	cur, err := r.coll.Find(ctx, bson.M{"submission_id": submissionID},
		options.Find().SetSort(bson.D{{Key: "round", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("list rejudges: %w", err)
	}
	defer cur.Close(ctx)
	out := make([]*model.Rejudge, 0)
	for cur.Next(ctx) {
		var rj model.Rejudge
		if err := cur.Decode(&rj); err != nil {
			return nil, fmt.Errorf("decode rejudge: %w", err)
		}
		out = append(out, &rj)
	}
	return out, nil
}
