package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/jackc/pgx/v5"
)

type SupervisorRepository interface {
	// WithTx mengembalikan salinan repository yang menjalankan seluruh
	// query-nya di dalam transaction tx.
	WithTx(tx DBTX) SupervisorRepository

	FindAll(ctx context.Context) ([]model.Supervisor, error)
	FindByID(ctx context.Context, id int) (model.Supervisor, error)
	FindByStudent(ctx context.Context, studentID int) ([]model.Supervisor, error)

	LockStudent(ctx context.Context, studentID int) error
	LockSupervisor(ctx context.Context, ids ...int) error

	InsertSupervision(ctx context.Context, supervisorID, studentID int) error
	IncrementSupervision(ctx context.Context, supervisorID int) (model.Supervisor, error)
	DecrementSupervision(ctx context.Context, supervisorID int) (model.Supervisor, error)
	DeleteSupervision(ctx context.Context, supervisorID, studentID int) error

	UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error)
}

const supervisorColumns = `id, name, quota, supervised_count, version, created_at`

type supervisorPGRepo struct {
	db DBTX
}

func NewSupervisorRepository(db DBTX) SupervisorRepository {
	return &supervisorPGRepo{db: db}
}

func (r *supervisorPGRepo) WithTx(tx DBTX) SupervisorRepository {
	return nil
}

func (r *supervisorPGRepo) FindAll(ctx context.Context) ([]model.Supervisor, error) {
	return r.collectSupervisor(
		r.db.Query(
			ctx,
			"SELECT "+supervisorColumns+" FROM supervisors ORDER BY id",
		),
	)
}
func (r *supervisorPGRepo) FindByID(ctx context.Context, id int) (model.Supervisor, error) {
	c, err := scanSupervisor(
		r.db.QueryRow(
			ctx,
			"SELECT "+supervisorColumns+" FROM supervisors WHERE id = $1",
			id,
		),
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrNotFound
		}

		return model.Supervisor{}, fmt.Errorf("mengambil supervisor: %w", err)
	}

	return c, nil
}

func (r *supervisorPGRepo) FindByStudent(ctx context.Context, studentID int) ([]model.Supervisor, error) {
	return r.collectSupervisor(
		r.db.Query(
			ctx,
			`SELECT c.id, c.name, c.quota,
			       c.supervised_count, c.version, c.created_at
			 FROM supervisor sr
			 JOIN supervisions ss ON ss.supervisor_id = sr.id
			 WHERE ss.student_id = $1
			 ORDER BY c.code`,
			studentID,
		),
	)
}

func (r *supervisorPGRepo) LockStudent(ctx context.Context, studentID int) error {
	var id int

	err := r.db.QueryRow(
		ctx,
		"SELECT id FROM students WHERE id = $1 FOR UPDATE",
		studentID,
	).Scan(&id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}

		return fmt.Errorf("mengunci data mahasiswa: %w", err)
	}

	return nil
}

func (r *supervisorPGRepo) LockSupervisor(ctx context.Context, ids ...int) error {
	rows, err := r.db.Query(
		ctx,
		"SELECT id FROM supervisor WHERE id = ANY($1) ORDER BY id FOR UPDATE",
		ids,
	)
	if err != nil {
		return fmt.Errorf("mengunci supervise=oe: %w", err)
	}

	defer rows.Close()

	found := 0

	for rows.Next() {
		found++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("mengunci supervisor: %w", err)
	}

	if found != len(ids) {
		return ErrNotFound
	}

	return nil
}

func (r *supervisorPGRepo) InsertSupervision(ctx context.Context, supervisorID, studentID int) error {
	_, err := r.db.Exec(ctx, "INSERT INTO supervisions (supervisor_id, student_id) VALUES ($1, $2)", supervisorID, studentID)

	if err != nil {
		if pgErrorCode(err) == codeUniqueViolation {
			return ErrDuplicate
		}
		return fmt.Errorf("mencatat pendaftaran bimbingan: %w", err)
	}

	return nil
}

func (r *supervisorPGRepo) IncrementSupervision(ctx context.Context, supervisorID int) (model.Supervisor, error) {
	c, err := scanSupervisor(
		r.db.QueryRow(
			ctx,
			`UPDATE supervisors
			 SET supervised_count = supervised_count + 1
			 WHERE id = $1
			   AND supervised_count < quota
			 RETURNING `+supervisorColumns,
			supervisorID,
		),
	)

	if err == nil {
		return c, nil
	}

	// Tidak ada row yang berubah. Dua kemungkinan: supervisor tidak ada,
	// atau supervisor ada tetapi sudah penuh. Bedakan keduanya.
	if _, findErr := r.FindByID(ctx, supervisorID); findErr != nil {
		return model.Supervisor{}, findErr
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("menambah peserta: %w", err)
	}

	return model.Supervisor{}, ErrCourseFull
}

func (r *supervisorPGRepo) DecrementSupervision(ctx context.Context, supervisorID int) (model.Supervisor, error) {
	c, err := scanSupervisor(
		r.db.QueryRow(
			ctx,
			`UPDATE supervisors
			 SET supervised_count = supervised_count - 1
			 WHERE id = $1
			 RETURNING `+supervisorColumns,
			supervisorID,
		),
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrNotFound
		}

		return model.Supervisor{}, fmt.Errorf("mengurangi peserta: %w", err)
	}

	return c, nil
}

func (r *supervisorPGRepo) DeleteSupervision(ctx context.Context, supervisorID, studentID int) error {
	tag, err := r.db.Exec(
		ctx,
		"DELETE FROM supervisions WHERE supervisor_id = $1 AND student_id = $2",
		supervisorID,
		studentID,
	)

	if err != nil {
		return fmt.Errorf("menghapus pendaftaran bimbingan: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotEnrolled
	}

	return nil
}

func (r *supervisorPGRepo) UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error) {
	c, err := scanSupervisor(
		r.db.QueryRow(
			ctx,
			`UPDATE supervisor
			 SET quota = $1, version = version + 1
			 WHERE id = $2
			   AND version = $3
			   AND $1 >= supervised_count
			 RETURNING `+supervisorColumns,
			quota,
			id,
			version,
		),
	)

	if err == nil {
		return c, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("mengubah kuota: %w", err)
	}

	current, findErr := r.FindByID(ctx, id)
	if findErr != nil {
		return model.Supervisor{}, findErr
	}

	if current.Version != version {
		return model.Supervisor{}, ErrVersionConflict
	}

	return model.Supervisor{}, ErrQuotaBelowFilled
}

func scanSupervisor(row pgx.Row) (model.Supervisor, error) {
	var c model.Supervisor

	err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Quota,
		&c.SupervisedCount,
		&c.Version,
		&c.CreatedAt,
	)

	return c, err
}

func (r *supervisorPGRepo) collectSupervisor(
	rows pgx.Rows,
	err error,
) ([]model.Supervisor, error) {
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar supervisor: %w", err)
	}

	defer rows.Close()

	result := []model.Supervisor{}

	for rows.Next() {
		c, err := scanSupervisor(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca row supervisor: %w", err)
		}

		result = append(result, c)
	}

	return result, rows.Err()
}
