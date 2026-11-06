package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrAlreadySubmit = errors.New("sudah pernah mengirim portofolio")
	ErrProjectsLimit = errors.New("jumlah daftar projek mencapai batas")
)

const (
	MaxProjectPerPortofolio = 20
)

type portofolioDocument struct {
	ID           bson.ObjectID         `bson:"_id,omitempty"`
	UserID       int                   `bson:"user_id"`
	StudentID    int                   `bson:"student_id"`
	Headline     string                `bson:"headline"`
	Bio          string                `bson:"bio"`
	Projects     []projectDocument     `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
	CreatedAt    time.Time             `bson:"created_at"`
	UpdatedAt    time.Time             `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
	CreatedAt    time.Time     `bson:"created_at"`
}

type certificateDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	Title     string        `bson:"title"`
	Publisher string        `bson:"publisher"`
	Date      time.Time     `bson:"date"`
}

func (d portofolioDocument) toModel() model.Portofolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		projects = append(projects, model.Project{
			ID:           p.ID.Hex(),
			Title:        p.Title,
			Technologies: p.Technologies,
			Link:         p.Link,
			Year:         p.Year,
		})
	}

	certificates := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certificates = append(certificates, model.Certificate{
			ID:        c.ID.Hex(),
			Title:     c.Title,
			Publisher: c.Title,
			Date:      c.Date,
		})
	}

	return model.Portofolio{
		ID:           d.ID.Hex(),
		UserID:       d.UserID,
		StudentID:    d.StudentID,
		Headline:     d.Headline,
		Bio:          d.Bio,
		Projects:     projects,
		Certificates: certificates,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

type portofolioMongoRepository struct {
	coll *mongo.Collection
}

type PortofolioRepository interface {
	EnsureIndexes(ctx context.Context) error

	Get(ctx context.Context, q model.PortofolioQuery) ([]model.Portofolio, error)
	Create(ctx context.Context, p model.Portofolio) (model.Portofolio, error)
	FindByID(ctx context.Context, id string) (model.Portofolio, error)
	UpdateOwn(ctx context.Context, id string, userID int, req model.UpdatePortofolioReq) (model.Portofolio, error)
	Delete(ctx context.Context, id string, userID int, anyOwner bool) error

	AddProject(ctx context.Context, id string, project model.Project) (model.Portofolio, error)
	TechSummary(ctx context.Context) (model.PortofolioTechSummary, error)
}

func NewPortofolioRepository(db *mongo.Database) PortofolioRepository {
	return &portofolioMongoRepository{
		coll: db.Collection("student_portofolios"),
	}
}

func (r *portofolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("portofolio_student_unique"),
		},
		{
			Keys:    bson.D{{Key: "projects.technologies", Value: 1}},
			Options: options.Index().SetName("technology_idx"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index portofolio: %w", err)
	}

	return nil
}

func (r *portofolioMongoRepository) Get(ctx context.Context, q model.PortofolioQuery) ([]model.Portofolio, error) {
	filter := bson.M{}

	if q.StudentID != nil {
		filter["student_id"] = *q.StudentID
	}

	if q.Before != "" {
		before, err := parseID(q.Before)
		if err != nil {
			return []model.Portofolio{}, err
		}
		filter["_id"] = bson.M{"$lt": before}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetLimit(int64(q.Limit))

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar portofolio: %w", err)
	}
	defer cursor.Close(ctx)

	var portofolioDocs []portofolioDocument
	if err := cursor.All(ctx, &portofolioDocs); err != nil {
		return nil, fmt.Errorf("membaca daftar portofolio: %w", err)
	}

	result := make([]model.Portofolio, 0, len(portofolioDocs))
	for _, d := range portofolioDocs {
		result = append(result, d.toModel())
	}

	return result, nil
}

func (r *portofolioMongoRepository) Create(ctx context.Context, p model.Portofolio) (model.Portofolio, error) {
	t := now()

	certificateDocs := make([]certificateDocument, 0, len(p.Certificates))
	for _, c := range p.Certificates {
		certificateDocs = append(certificateDocs, certificateDocument{
			ID:        bson.NewObjectID(),
			Title:     c.Title,
			Publisher: c.Publisher,
			Date:      c.Date,
		})
	}

	portoDoc := portofolioDocument{
		ID:           bson.NewObjectID(),
		UserID:       p.UserID,
		StudentID:    p.StudentID,
		Headline:     p.Headline,
		Bio:          p.Bio,
		Projects:     []projectDocument{},
		Certificates: certificateDocs,
		CreatedAt:    t,
		UpdatedAt:    t,
	}
	if portoDoc.Certificates == nil {
		portoDoc.Certificates = []certificateDocument{}
	}

	if _, err := r.coll.InsertOne(ctx, portoDoc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.Portofolio{}, ErrDuplicate
		}

		return model.Portofolio{}, fmt.Errorf("menyimpan portofolio: %w", err)
	}

	return portoDoc.toModel(), nil
}

func (r *portofolioMongoRepository) FindByID(ctx context.Context, id string) (model.Portofolio, error) {
	oid, err := parseID(id)
	if err != nil {
		return model.Portofolio{}, err
	}

	var portofolioDoc portofolioDocument
	if err := r.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&portofolioDoc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portofolio{}, ErrNotFound
		}
		return model.Portofolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}

	return portofolioDoc.toModel(), nil
}

func (r *portofolioMongoRepository) UpdateOwn(ctx context.Context, id string, userID int, req model.UpdatePortofolioReq) (model.Portofolio, error) {
	oid, err := parseID(id)
	if err != nil {
		return model.Portofolio{}, err
	}

	set := bson.M{
		"updated_at": now(),
		"bio":        req.Bio,
		"headline":   req.Headline,
	}

	certificateDocs := make([]certificateDocument, 0, len(req.Certificates))
	for _, c := range req.Certificates {
		certificateDocs = append(certificateDocs, certificateDocument{
			ID:        bson.NewObjectID(),
			Title:     c.Title,
			Publisher: c.Publisher,
			Date:      c.Date,
		})
	}

	if certificateDocs == nil {
		certificateDocs = []certificateDocument{}
	}

	set["certificates"] = certificateDocs

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var portofolioDoc portofolioDocument
	err = r.coll.FindOneAndUpdate(ctx,
		bson.M{"_id": oid, "user_id": userID},
		bson.M{"$set": set},
		opts,
	).Decode(&portofolioDoc)

	if err == nil {
		return portofolioDoc.toModel(), nil
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portofolio{}, fmt.Errorf("mengubah portofolio: %w", err)
	}

	return model.Portofolio{}, r.whyNoMatchPorto(ctx, oid)
}

func (r *portofolioMongoRepository) Delete(ctx context.Context, id string, userID int, anyOwner bool) error {
	oid, err := parseID(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": oid}
	if !anyOwner {
		filter["user_id"] = userID
	}

	res, err := r.coll.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("menghapus portofolio: %w", err)
	}
	if res.DeletedCount == 0 {
		return r.whyNoMatchPorto(ctx, oid)
	}

	return nil
}

func (r *portofolioMongoRepository) AddProject(ctx context.Context, id string, project model.Project) (model.Portofolio, error) {
	oid, err := parseID(id)
	if err != nil {
		return model.Portofolio{}, err
	}

	projectDoc := projectDocument{
		ID:           bson.NewObjectID(),
		Title:        project.Title,
		Technologies: project.Technologies,
		Link:         project.Link,
		Year:         project.Year,
		CreatedAt:    now(),
	}
	limitField := "projects." + strconv.Itoa(MaxProjectPerPortofolio-1)

	var updated portofolioDocument
	err = r.coll.FindOneAndUpdate(ctx,
		bson.M{"_id": oid, limitField: bson.M{"$exists": false}},
		bson.M{"$push": bson.M{"projects": projectDoc}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)

	if err == nil {
		return updated.toModel(), nil
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portofolio{}, fmt.Errorf("menambah balasan: %w", err)
	}

	count, err := r.coll.CountDocuments(ctx, bson.M{"_id": oid})
	if err != nil {
		return model.Portofolio{}, fmt.Errorf("memeriksa ulasan: %w", err)
	}

	if count == 0 {
		return model.Portofolio{}, ErrNotFound
	}

	return model.Portofolio{}, ErrProjectsLimit
}

func (r *portofolioMongoRepository) TechSummary(ctx context.Context) (model.PortofolioTechSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$facet", Value: bson.M{
			"stats": bson.A{
				bson.M{
					"$count": "count",
				},
			},
			"top_technologies": bson.A{
				bson.M{"$unwind": "$projects"},
				bson.M{"$unwind": "$projects.technologies"},
				bson.M{"$group": bson.M{
					"_id":   "$projects.technologies",
					"total": bson.M{"$sum": 1},
				}},
				bson.M{"$sort": bson.D{
					{Key: "total", Value: -1},
					{Key: "_id", Value: 1},
				}},
				bson.M{"$limit": 10},
			},
		}}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return model.PortofolioTechSummary{}, fmt.Errorf("meringkas ulasan: %w", err)
	}
	defer cursor.Close(ctx)

	var out []struct {
		Stats []struct {
			Count int `bson:"count"`
		} `bson:"stats"`

		TopTech []struct {
			Tech  string `bson:"_id"`
			Total int    `bson:"total"`
		} `bson:"top_technologies"`
	}
	if err := cursor.All(ctx, &out); err != nil {
		return model.PortofolioTechSummary{}, fmt.Errorf("membaca ringkasan: %w", err)
	}

	techSummary := model.PortofolioTechSummary{}
	if len(out) == 0 {
		return techSummary, nil
	}

	if len(out[0].Stats) > 0 {
		techSummary.Count = out[0].Stats[0].Count
	}

	for _, d := range out[0].TopTech {
		techSummary.TopTechnologies = append(techSummary.TopTechnologies, model.TechCount{Technology: d.Tech, Total: d.Total})
	}

	return techSummary, nil
}

func (r *portofolioMongoRepository) whyNoMatchPorto(ctx context.Context, oid bson.ObjectID) error {
	count, err := r.coll.CountDocuments(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("memeriksa portofolio: %w", err)
	}

	if count == 0 {
		return ErrNotFound
	}

	return ErrNotOwner
}
