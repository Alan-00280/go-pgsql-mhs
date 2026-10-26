package repository

import "errors"

// Setinel Errs
var (
	ErrNotFound         = errors.New("data not found")
	ErrDuplicate        = errors.New("data already exists")
	ErrNotEnrolled      = errors.New("tidak terdaftar di kelas ini")
	ErrCourseFull       = errors.New("kuota kelas sudah penuh")
	ErrVersionConflict  = errors.New("data sudah diubah pihak lain")
	ErrQuotaBelowFilled = errors.New("kuota lebih kecil dari jumlah peserta")
)
