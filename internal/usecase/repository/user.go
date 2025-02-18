package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"unicode/utf8"

	tx "github.com/Thiht/transactor/pgx"

	"github.com/jackc/pgx/v5"

	"clean-arch-template/internal/entity"
)

var (
	ErrInvalidInputData      = errors.New("invalid input data")
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrNegativeAmount        = errors.New("transfer amount must be positive")
	ErrSameAccount           = errors.New("cannot transfer to the same account")
	ErrSourceAccountNotFound = errors.New("source account not found")
	ErrDestAccountNotFound   = errors.New("destination account not found")
)

type UserRepository struct {
	db         tx.DBGetter
	transactor *tx.Transactor
}

func NewUserRepository(db tx.DBGetter, transactor *tx.Transactor) *UserRepository {
	repo := &UserRepository{
		db:         db,
		transactor: transactor,
	}

	return repo
}

func (r *UserRepository) GetAllUsers(ctx context.Context, offset, limit int) ([]entity.User, error) {
	query := `
		SELECT u.id,
		       u.name
		FROM users u
		LEFT JOIN orders o ON u.id = o.user_id
		GROUP BY u.id, u.name
		ORDER BY u.id
		OFFSET $1 LIMIT $2
	`

	raw, err := r.db(ctx).Query(ctx, query, offset, limit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	users, err := pgx.CollectRows(raw, pgx.RowToStructByName[entity.User])
	if err != nil {
		slog.Error("failed to collect rows", "error", err)
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) GetAllUsersWithOrders(ctx context.Context, offset, limit int) ([]entity.UserOrders, error) {
	query := `
		SELECT u.id,
		       u.name,
		       COALESCE(array_agg(o.id) FILTER (WHERE o.id IS NOT NULL), '{}') as order_ids,
		       COALESCE(array_agg(o.amount) FILTER (WHERE o.id IS NOT NULL), '{}') as order_amounts
		FROM users u
		LEFT JOIN orders o ON u.id = o.user_id
		GROUP BY u.id, u.name
		ORDER BY u.id
		OFFSET $1 LIMIT $2
	`

	raw, err := r.db(ctx).Query(ctx, query, offset, limit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	type row struct {
		ID           int     `db:"id"`
		Name         string  `db:"name"`
		OrderIDs     []int64 `db:"order_ids"`
		OrderAmounts []int64 `db:"order_amounts"`
	}

	rows, err := pgx.CollectRows(raw, pgx.RowToStructByName[row])

	if err != nil {
		return nil, fmt.Errorf("failed to collect rows: %w", err)
	}

	result := make([]entity.UserOrders, 0)

	for _, r := range rows {
		user := entity.UserOrders{
			ID:     r.ID,
			Name:   r.Name,
			Orders: make([]entity.Order, 0, len(r.OrderIDs)),
		}

		for i, orderID := range r.OrderIDs {
			// Ensure arrays lengths match
			if i < len(r.OrderAmounts) {
				order := entity.Order{
					ID:     orderID,
					UserID: int64(r.ID),
					Amount: r.OrderAmounts[i],
				}
				user.Orders = append(user.Orders, order)
			}
		}

		result = append(result, user)
	}

	return result, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*entity.User, error) {
	query := `
		SELECT u.id, u.name
		FROM users u
		WHERE u.id=$1
		GROUP BY u.id, u.name
	`

	var user entity.User

	err := r.db(ctx).QueryRow(ctx, query, id).Scan(&user.ID, &user.Name)
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

func (r *UserRepository) TransferMoney(ctx context.Context, transfer entity.Transfer) error {
	if transfer.FromAccountID == transfer.ToAccountID {
		return ErrSameAccount
	}

	query := `SELECT balance FROM users WHERE id = $1`

	return r.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		var sourceBalance float64

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
		err = r.db(ctx).QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", transfer.ToAccountID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check destination account: %w", err)
		}

		if !exists {
			return ErrDestAccountNotFound
		}

		_, err = r.db(ctx).Exec(ctx, `
			UPDATE users 
			SET balance = balance - $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, transfer.Amount, transfer.FromAccountID)
		if err != nil {
			return fmt.Errorf("failed to update source account: %w", err)
		}

		_, err = r.db(ctx).Exec(ctx, `
			UPDATE users 
			SET balance = balance + $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, transfer.Amount, transfer.ToAccountID)
		if err != nil {
			return fmt.Errorf("failed to update destination account: %w", err)
		}

		_, err = r.db(ctx).Exec(ctx, `
			INSERT INTO transactions(from_user_id, to_user_id, amount)
			VALUES($1, $2, $3)
		`, transfer.FromAccountID, transfer.ToAccountID, transfer.Amount)
		if err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		return nil
	})
}
