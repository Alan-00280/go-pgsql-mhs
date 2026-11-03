package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrInvalidID    = errors.New("format id tidak valid")
	ErrNotOwner     = errors.New("bukan pemilik data")
	ErrOwnReview    = errors.New("tidak boleh menilai ulasan sendiri")
	ErrAlreadyVoted = errors.New("sudah pernah menilai ulasan ini")
	ErrReplyLimit   = errors.New("jumlah balasan sudah mencapai batas")
)

// konstanta batas pertumbuhan array replies
const MaxRepliesPerReview = 50

type reviewMongoRepository struct {
	coll *mongo.Collection
}

type ReviewRepository interface {
	EnsureIndexes(ctx context.Context) error

	Create(ctx context.Context, r model.Review) (model.Review, error)
	FindByID(ctx context.Context, id string) (model.Review, error)
	FindByCourse(ctx context.Context, q model.ReviewQuery) ([]model.Review, error)
	UpdateOwn(ctx context.Context, id string, userID int, req model.UpdateReviewRequest) (model.Review, error)
	Delete(ctx context.Context, id string, userID int, anyOwner bool) error

	AddReply(ctx context.Context, id string, reply model.Reply) (model.Review, error)
	MarkHelpful(ctx context.Context, id string, userID int) (model.Review, error)
	Summary(ctx context.Context, courseID int) (model.ReviewSummary, error)
}

func NewReviewRepository(db *mongo.Database) ReviewRepository {
	return &reviewMongoRepository{coll: db.Collection("course_reviews")}
}

// Struct untuk mendefinisikan bentuk yang SEBAGAIMANA TERSIMPAN di MongoDB
// json tidak diperlukan karena bentuk struct ini tidak dikirim ke client
type reviewDocument struct {
	ID            bson.ObjectID   `bson:"_id,omitempty"`
	CourseID      int             `bson:"course_id"`
	UserID        int             `bson:"user_id"`
	Username      string          `bson:"username"`
	Rating        int             `bson:"rating"`
	Comment       string          `bson:"comment"`
	Tags          []string        `bson:"tags"`
	Replies       []replyDocument `bson:"replies"`
	HelpfullCount int             `bson:"helpfull_count"`
	HelpfullBy    []int           `bson:"helpfull_by"`
	CreatedAt     time.Time       `bson:"created_at"`
	UpdatedAt     time.Time       `bson:"updated_at"`
}

type replyDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	UserID    int           `bson:"user_id"`
	Username  string        `bson:"username"`
	Body      string        `bson:"body"`
	CreatedAt time.Time     `bson:"created_at"`
}

func (d reviewDocument) toModel() model.Review {
	replies := make([]model.Reply, 0, len(d.Replies))
	for _, r := range d.Replies {
		replies = append(replies, model.Reply{
			ID:        r.ID.Hex(),
			UserID:    r.UserID,
			Username:  r.Username,
			Body:      r.Body,
			CreatedAt: r.CreatedAt,
		})
	}

	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}

	return model.Review{
		ID:           d.ID.Hex(),
		CourseID:     d.CourseID,
		UserID:       d.UserID,
		Username:     d.Username,
		Rating:       d.Rating,
		Comment:      d.Comment,
		Tags:         tags,
		Replies:      replies,
		HelpfulCount: d.HelpfullCount,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

func now() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

func parseID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, ErrInvalidID
	}

	return oid, nil
}

func (r *reviewMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// Satu mahasiswa Satu ulasan per kelas
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("course_user_unique"),
		},
		{
			// daftar ulasan satu kelas dengan terbaru lebih dulu
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "_id", Value: -1}},
			Options: options.Index().SetName("course_newest"),
		},
		{
			// filter berdasarkan tag
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "tags", Value: 1}},
			Options: options.Index().SetName("course_tag"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index course_reviews: %w", err)
	}

	return nil
}

func (r *reviewMongoRepository) Create(ctx context.Context, review model.Review) (model.Review, error) {
	t := now()
	doc := reviewDocument{
		ID:         bson.NewObjectID(),
		CourseID:   review.CourseID,
		UserID:     review.UserID,
		Username:   review.Username,
		Rating:     review.Rating,
		Comment:    review.Comment,
		Tags:       review.Tags,
		Replies:    []replyDocument{},
		HelpfullBy: []int{},
		CreatedAt:  t,
		UpdatedAt:  t,
	}
	if doc.Tags == nil {
		doc.Tags = []string{}
	}

	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.Review{}, ErrDuplicate
		}

		return model.Review{}, fmt.Errorf("menyimpan ulasan: %w", err)
	}

	return doc.toModel(), nil
}
func (r *reviewMongoRepository) FindByID(ctx context.Context, id string) (model.Review, error) {
	oid, err := parseID(id)
	if err != nil {
		return model.Review{}, err
	}

	var doc reviewDocument
	if err := r.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Review{}, ErrNotFound
		}
		return model.Review{}, fmt.Errorf("mengambil ulasan: %w", err)
	}

	return doc.toModel(), nil
}
func (r *reviewMongoRepository) FindByCourse(ctx context.Context, q model.ReviewQuery) ([]model.Review, error) {
	filter := bson.M{"course_id": q.CourseID}

	if q.Tag != "" {
		filter["tags"] = q.Tag
	}

	if q.Before != "" {
		before, err := parseID(q.Before)
		if err != nil {
			return []model.Review{}, err
		}
		filter["_id"] = bson.M{"$lt": before}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetLimit(int64(q.Limit)).
		SetProjection(bson.M{"helpfull_by": 0})

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar ulasan: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []reviewDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("membaca daftar ulasan: %w", err)
	}

	result := make([]model.Review, 0, len(docs))
	for _, d := range docs {
		result = append(result, d.toModel())
	}

	return result, nil
}
func (r *reviewMongoRepository) UpdateOwn(ctx context.Context, id string, userID int, req model.UpdateReviewRequest) (model.Review, error) {
	return model.Review{}, nil
}
func (r *reviewMongoRepository) Delete(ctx context.Context, id string, userID int, anyOwner bool) error {
	return nil
}

func (r *reviewMongoRepository) AddReply(ctx context.Context, id string, reply model.Reply) (model.Review, error) {
	return model.Review{}, nil
}
func (r *reviewMongoRepository) MarkHelpful(ctx context.Context, id string, userID int) (model.Review, error) {
	return model.Review{}, nil
}
func (r *reviewMongoRepository) Summary(ctx context.Context, courseID int) (model.ReviewSummary, error) {
	return model.ReviewSummary{}, nil
}
