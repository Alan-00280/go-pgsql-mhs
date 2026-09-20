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
}

var sortColumnUser = map[string]string{
	"id":         "id",
	"username":   "username",
	"email":      "email",
	"created_at": "created_at",
}

// NEW filter role
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
	if q.Order != "desc" {
		direction = "DESC"
	}

	sqlText := fmt.Sprintf(
		`SELECT id, username, email, role, is_active, created_at FROM users %s ORDER BY %s %s LIMIT $%d OFFSET $%d`, where, sortColumnUser[q.Order], direction, len(args)+1, len(args)+2,
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

	if err := r.pool.QueryRow(ctx, "SELECT id, username, email, role, is_active, created_at FROM users WHERE id = $1", id).Scan(&result.ID, &result.Username, &result.Email, &result.Role, &result.IsActive, &result.CreatedAt); err != nil {
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
		`SELECT id, username, email, password, role, is_active, created_at
         FROM users WHERE LOWER(username) = LOWER($1)`, username,
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
