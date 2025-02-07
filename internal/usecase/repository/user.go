package repository

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"clean-arch-template/internal/entity"
	"clean-arch-template/pkg/database"
)

var ErrInvalidInputData = errors.New("invalid input data")

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
	// Инициализируем карту пользователей и слайс для сохранения порядка
	userMap := make(map[int]*entity.User)
	var users []*entity.User

	query := `
		SELECT u.id,
			   u.name,
			   o.id     as order_id,
			   o.amount as order_amount
		FROM users u
   		LEFT JOIN orders o ON u.id = o.user_id
		ORDER BY u.id, o.id
		OFFSET $1 LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, offset, limit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var (
		userID      int
		userName    string
		orderID     pgtype.Int4
		orderAmount pgtype.Int4
	)

	_, err = pgx.ForEachRow(rows, []any{&userID, &userName, &orderID, &orderAmount}, func() error {
		// Проверяем, существует ли пользователь в map
		var user *entity.User

		user, exists := userMap[userID]
		if !exists {
			user = &entity.User{
				ID:     userID,
				Name:   userName,
				Orders: make([]entity.Order, 0),
			}
			userMap[userID] = user
			users = append(users, user)
		}

		err = fillUserOrders(user, orderID, orderAmount)
		if err != nil {
			return err
		}

		return nil
	})

	// Преобразуем слайс указателей на пользователей в слайс значений
	result := make([]entity.User, len(users))
	for i, u := range users {
		result[i] = *u
	}

	return result, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*entity.User, error) {
	var user entity.User
	var found bool

	var (
		orderID     pgtype.Int4
		orderAmount pgtype.Int4
	)

	query := `
		SELECT u.id, 
			   u.name, 
			   o.id as order_id, 
			   o.amount as order_amount 
		FROM users u LEFT JOIN orders o ON u.id = o.user_id 
	   WHERE u.id=$1
	`

	rows, err := r.db.Query(ctx, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	_, err = pgx.ForEachRow(rows, []any{&user.ID, &user.Name, &orderID, &orderAmount}, func() error {
		found = true
		err = fillUserOrders(&user, orderID, orderAmount)
		if err != nil {
			return err
		}

		return nil
	})

	if !found {
		return nil, nil
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
	_, err := r.db.Exec(ctx, "UPDATE users SET name = $2 WHERE id = $1", input.ID, input.Name)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, input *entity.User) error {
	_, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", input.ID)
	if err != nil {
		return err
	}

	return nil
}

func fillUserOrders(user *entity.User, orderID pgtype.Int4, orderAmount pgtype.Int4) error {
	// Если заказ не NULL, обрабатываем его
	if orderID.Valid {
		id, err := orderID.Int64Value()
		if err != nil {
			return err
		}
		// Ищем или добавляем заказ у пользователя
		var order *entity.Order
		orderFound := false

		for i := range user.Orders {
			if user.Orders[i].ID == id.Int64 {
				order = &user.Orders[i]
				orderFound = true
				break
			}
		}

		if !orderFound {
			amount, e := orderAmount.Int64Value()
			if e != nil {
				return e
			}

			order = &entity.Order{
				ID:     id.Int64,
				UserID: int64(user.ID),
				Amount: amount.Int64,
			}
		}

		user.Orders = append(user.Orders, *order)
	}

	return nil
}
