package repository

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Setinel Errs
var (
	ErrNotFound         = errors.New("data not found")
	ErrDuplicate        = errors.New("data already exists")
	ErrNotEnrolled      = errors.New("tidak terdaftar di kelas ini")
	ErrCourseFull       = errors.New("kuota kelas sudah penuh")
	ErrVersionConflict  = errors.New("data sudah diubah pihak lain")
	ErrQuotaBelowFilled = errors.New("kuota lebih kecil dari jumlah peserta")
)

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
