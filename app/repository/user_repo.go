package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.User, int, error)
	FindByID(ctx context.Context, id int) (model.User, error)
	FindByUsername(ctx context.Context, username string) (model.User, error)
	Create(ctx context.Context, u model.User) (model.User, error)
	Update(ctx context.Context, u model.User) (model.User, error)
	Delete(ctx context.Context, id int) error
	UpdateRole(ctx context.Context, id int, role string) (model.User, error)
	FindAfterCursor(ctx context.Context, q model.CursorQuery) ([]model.User, error)
}

var sortColumnUser = map[string]string{
	"id":         "id",
	"username":   "username",
	"email":      "email",
	"created_at": "created_at",
}

var userColumns string = "id, username, email, role, is_active, created_at"

func buildFilterUser(q model.ListQuery) (string, []any) {
	where := " WHERE 1=1"
	args := []any{}

	if q.Search != "" {
		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)",
			len(args)+1, len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}

	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}

	if q.UserFilter != nil {
		if q.UserFilter.Role != "" {
			where += fmt.Sprintf(" AND role = $%d", len(args)+1)
			args = append(args, q.UserFilter.Role)
		}
	}

	return where, args
}

type UserPGRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &UserPGRepository{pool: pool}
}

func (r *UserPGRepository) FindAll(ctx context.Context, q model.ListQuery) ([]model.User, int, error) {
	where, args := buildFilterUser(q)

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("[ERROR] count total user: %w", err)
	}

	direction := "ASC"
	if q.Order != "asc" {
		direction = "DESC"
	}

	sqlText := fmt.Sprintf(
		`SELECT %s FROM users %s ORDER BY %s %s LIMIT $%d OFFSET $%d`, userColumns, where, sortColumnUser[q.Sort], direction, len(args)+1, len(args)+2,
	)
	args = append(args, q.Limit, q.Offset())

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("[ERROR] can't get rows from users: %w", err)
	}

	result := []model.User{}
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("[ERROR] can't scan rows from users: %w", err)
		}
		result = append(result, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("[ERROR] error query from users: %w", err)
	}

	return result, 0, nil
}

func (r *UserPGRepository) FindByID(ctx context.Context, id int) (model.User, error) {
	result := model.User{}

	if err := r.pool.QueryRow(ctx, "SELECT %s FROM users WHERE id = $1", userColumns, id).Scan(&result.ID, &result.Username, &result.Email, &result.Role, &result.IsActive, &result.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}

		return model.User{}, fmt.Errorf("[ERROR] can't get from users: %w", err)
	}

	return result, nil
}

func (r *UserPGRepository) Create(ctx context.Context, u model.User) (model.User, error) {
	if err := r.pool.QueryRow(ctx, "INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id, created_at", u.Username, u.Email, u.Password, u.Role).Scan(&u.ID, &u.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return model.User{}, ErrDuplicate
		}

		return model.User{}, fmt.Errorf("[ERROR] can't create user: %w", err)
	}

	return u, nil
}

func (r *UserPGRepository) Update(ctx context.Context, u model.User) (model.User, error) {
	if err := r.pool.QueryRow(ctx, "UPDATE users SET username = $1, email = $2, is_active = $3 WHERE id = $4 RETURNING id, username, email, is_active, created_at", u.Username, u.Email, u.IsActive, u.ID).Scan(&u.ID, &u.Username, &u.Email, &u.IsActive, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}

		return model.User{}, fmt.Errorf("[ERROR] can't update user: %w", err)
	}

	return u, nil
}

func (r *UserPGRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("[ERROR] can't delete user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *UserPGRepository) FindByUsername(
	ctx context.Context, username string,
) (model.User, error) {
	var u model.User

	err := r.pool.QueryRow(ctx,
		`SELECT %s
         FROM users WHERE LOWER(username) = LOWER($1)`, userColumns, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role,
		&u.IsActive, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}

	return u, nil
}

func (r *UserPGRepository) UpdateRole(
	ctx context.Context, id int, role string,
) (model.User, error) {
	var u model.User

	if err := r.pool.QueryRow(ctx, `UPDATE users SET role = $1 WHERE id = $2 RETURNING id, username, email, role, is_active, created_at `, role, id).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}

		return model.User{}, fmt.Errorf("[ERROR] can't update role user: %w", err)
	}

	return u, nil
}

// FindAfterCursor mengambil satu halaman memakai keyset pagination.
//
// id ikut dibandingkan karena created_at TIDAK dijamin unik. Bila dua
// baris dibuat pada mikrodetik yang sama dan hanya created_at yang
// dibandingkan, salah satu baris akan terlewat atau terkirim dua kali.
// Jumlah yang diminta sengaja limit+1. Baris tambahan itu tidak dikirim
// ke client; keberadaannya hanya dipakai untuk menjawab "masih ada
// halaman berikutnya?" tanpa perlu COUNT(*) atas seluruh tabel.
func (r *UserPGRepository) FindAfterCursor(
	ctx context.Context, q model.CursorQuery,
) ([]model.User, error) {
	args := []any{}
	where := " WHERE 1=1 "

	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		where += fmt.Sprintf(" AND username ILIKE $%d", len(args))
	}
	if q.IsActive != nil {
		args = append(args, *q.IsActive)
		where += fmt.Sprintf(" AND is_active = $%d", len(args))
	}
	if q.After != nil {
		args = append(args, q.After.CreatedAt, q.After.ID)
		where += fmt.Sprintf(" AND (created_at, id) < (%d, %d)", len(args)-1, len(args))
	}

	args = append(args, q.Limit+1)
	query := fmt.Sprintf("SELECT %s FROM users%s ORDER BY created_at DESC, id DESC LIMIT $%d", userColumns, where, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("can't get users: %w", err)
	}
	defer rows.Close()

	result := []model.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("can't read user: %w", err)
		}
		result = append(result, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("can't read user: %w", err)
	}

	return result, nil
}

func scanUser(rows pgx.Rows) (model.User, error) {
	var u model.User

	if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
		return model.User{}, err
	}

	return u, nil
}
