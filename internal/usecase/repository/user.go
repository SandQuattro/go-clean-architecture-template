package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"unicode/utf8"

	tx "github.com/Thiht/transactor/pgx"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"clean-arch-template/internal/entity"
)

var (
	ErrInvalidInputData      = errors.New("invalid input data")
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrAccountNotFound       = errors.New("account not found")
	ErrNegativeAmount        = errors.New("transfer amount must be positive")
	ErrSameAccount           = errors.New("cannot transfer to the same account")
	ErrDestAccountNotFound   = errors.New("destination account not found")
	ErrSourceAccountNotFound = errors.New("source account not found")
)

type UserRepository struct {
	db         tx.DBGetter
	transactor *tx.Transactor
}

func NewUserRepository(once *sync.Once, db tx.DBGetter, transactor *tx.Transactor) *UserRepository {
	var repo *UserRepository
	once.Do(func() {
		repo = &UserRepository{
			db:         db,
			transactor: transactor,
		}
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

	rows, err := r.db(ctx).Query(ctx, query, offset, limit)
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

	rows, err := r.db(ctx).Query(ctx, query, id)
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
	err := r.db(ctx).QueryRow(ctx, "INSERT INTO users(name) VALUES($1) RETURNING id", input.Name).Scan(&userID)
	if err != nil {
		return nil, err
	}

	input.ID = userID

	return input, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, input *entity.User) (*entity.User, error) {
	_, err := r.db(ctx).Exec(ctx, "UPDATE users SET name = $2 WHERE id = $1", input.ID, input.Name)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, input *entity.User) error {
	_, err := r.db(ctx).Exec(ctx, "DELETE FROM users WHERE id = $1", input.ID)
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

func (r *UserRepository) TransferMoney(ctx context.Context, transfer entity.Transfer) error {
	if transfer.FromAccountID == transfer.ToAccountID {
		return ErrSameAccount
	}

	return r.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		var sourceBalance float64

		query := `SELECT balance FROM transactions WHERE id = $1`
		if tx.IsWithinTransaction(ctx) {
			query += ` FOR UPDATE`
		}

		err := r.db(ctx).QueryRow(ctx, query, transfer.FromAccountID).Scan(&sourceBalance)

		if errors.Is(err, sql.ErrNoRows) {
			return ErrSourceAccountNotFound
		}

		if err != nil {
			return fmt.Errorf("failed to get source account balance: %w", err)
		}

		if transfer.Amount <= 0 {
			return ErrNegativeAmount
		}

		if sourceBalance < transfer.Amount {
			return ErrInsufficientFunds
		}

		var exists bool
		err = r.db(ctx).QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM transactions WHERE id = $1)", transfer.ToAccountID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check destination account: %w", err)
		}

		if !exists {
			return ErrDestAccountNotFound
		}

		_, err = r.db(ctx).Exec(ctx, `
			UPDATE transactions 
			SET balance = balance - $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, transfer.Amount, transfer.FromAccountID)

		if err != nil {
			return fmt.Errorf("failed to update source account: %w", err)
		}

		_, err = r.db(ctx).Exec(ctx, `
			UPDATE transactions 
			SET balance = balance + $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, transfer.Amount, transfer.ToAccountID)

		if err != nil {
			return fmt.Errorf("failed to update destination account: %w", err)
		}

		return nil
	})
}
