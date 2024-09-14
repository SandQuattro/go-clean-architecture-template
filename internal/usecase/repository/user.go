package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"sync"
	"unicode/utf8"

	"clean-arch-template/internal/entity"
	"clean-arch-template/pkg/database"
)

var (
	ErrInvalidInputData = errors.New("invalid input data")
)

type UserRepository struct {
	db database.Database
}

func NewUserRepository(once *sync.Once, db database.Database) *UserRepository {
	var repo *UserRepository
	once.Do(func() {
		repo = &UserRepository{db: db}
	})

	return repo
}

func (r *UserRepository) GetAllUsers(ctx context.Context, offset, limit int) ([]entity.User, error) {
	users := make([]entity.User, 0)

	rows, err := r.db.Query(ctx, "SELECT id, name FROM users OFFSET $1 LIMIT $2", offset, limit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var user entity.User
	for rows.Next() {
		err = rows.Scan(&user.ID, &user.Name)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*entity.User, error) {
	var user entity.User
	err := r.db.QueryRow(ctx, "SELECT id, name FROM users WHERE id=$1", id).Scan(&user.ID, &user.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) InsertUser(ctx context.Context, input *entity.User) (*entity.User, error) {
	if input.Name == "" {
		return nil, ErrInvalidInputData
	}

	// Проверка валидности UTF-8 строки
	if !utf8.ValidString(input.Name) {
		slog.Error("name is invalid")
		return nil, ErrInvalidInputData
	}

	var userID int

	slog.Debug("Inserting user with name", "name", input.Name)
	err := r.db.QueryRow(ctx, "INSERT INTO users(name) VALUES($1) RETURNING id", input.Name).Scan(&userID)
	if err != nil {
		return nil, err
	}

	input.ID = userID

	return input, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, input *entity.User) (*entity.User, error) {
	var user entity.User
	_, err := r.db.Exec(ctx, "UPDATE users SET name = $2 WHERE id = $1", input.ID, input.Name)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, input *entity.User) error {
	_, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", input.ID)
	if err != nil {
		return err
	}

	return nil
}
