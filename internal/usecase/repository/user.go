package repository

import (
	"clean-arch-template/internal/entity"
	"context"
	"errors"
	"fmt"

	tx "github.com/Thiht/transactor/pgx"

	"github.com/jackc/pgx/v5"
)

// Transactor запускает функцию внутри транзакции БД; текущая транзакция
// прокидывается через контекст (см. github.com/Thiht/transactor).
type Transactor interface {
	WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error
}

type UserRepository struct {
	db         tx.DBGetter
	transactor Transactor
}

func NewUserRepository(db tx.DBGetter, transactor Transactor) *UserRepository {
	return &UserRepository{
		db:         db,
		transactor: transactor,
	}
}

func (r *UserRepository) GetAllUsers(ctx context.Context, offset, limit int) ([]entity.User, error) {
	query := `
		SELECT u.id,
		       u.name
		FROM users u
		ORDER BY u.id
		OFFSET $1 LIMIT $2
	`

	raw, err := r.db(ctx).Query(ctx, query, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}

	users, err := pgx.CollectRows(raw, pgx.RowToStructByName[entity.User])
	if err != nil {
		return nil, fmt.Errorf("collect users: %w", err)
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
	if err != nil {
		return nil, fmt.Errorf("query users with orders: %w", err)
	}

	type row struct {
		ID           int     `db:"id"`
		Name         string  `db:"name"`
		OrderIDs     []int64 `db:"order_ids"`
		OrderAmounts []int64 `db:"order_amounts"`
	}

	rows, err := pgx.CollectRows(raw, pgx.RowToStructByName[row])
	if err != nil {
		return nil, fmt.Errorf("collect users with orders: %w", err)
	}

	result := make([]entity.UserOrders, 0, len(rows))

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
		WHERE u.id = $1
	`

	var user entity.User

	err := r.db(ctx).QueryRow(ctx, query, id).Scan(&user.ID, &user.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, entity.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by id: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) InsertUser(ctx context.Context, input *entity.User) (*entity.User, error) {
	err := r.db(ctx).QueryRow(ctx, "INSERT INTO users(name) VALUES($1) RETURNING id", input.Name).Scan(&input.ID)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return input, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, input *entity.User) (*entity.User, error) {
	// Одним запросом, без предварительного чтения: RETURNING отличает
	// «обновлено» от «не найдено» атомарно.
	err := r.db(ctx).
		QueryRow(ctx, "UPDATE users SET name = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING id, name", input.ID, input.Name).
		Scan(&input.ID, &input.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, entity.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return input, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id int) error {
	ct, err := r.db(ctx).Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return entity.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) TransferMoney(ctx context.Context, transfer entity.Transfer) error {
	return r.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		// Блокируем обе строки одним запросом в детерминированном порядке (ORDER BY id),
		// иначе встречные переводы A→B и B→A взаимно блокируются (deadlock).
		raw, err := r.db(ctx).Query(ctx,
			"SELECT id, balance FROM users WHERE id = ANY($1) ORDER BY id FOR UPDATE",
			[]int64{transfer.FromAccountID, transfer.ToAccountID},
		)
		if err != nil {
			return fmt.Errorf("lock accounts: %w", err)
		}

		type account struct {
			ID      int64 `db:"id"`
			Balance int64 `db:"balance"`
		}

		accounts, err := pgx.CollectRows(raw, pgx.RowToStructByName[account])
		if err != nil {
			return fmt.Errorf("collect accounts: %w", err)
		}

		balances := make(map[int64]int64, len(accounts))
		for _, acc := range accounts {
			balances[acc.ID] = acc.Balance
		}

		sourceBalance, ok := balances[transfer.FromAccountID]
		if !ok {
			return entity.ErrSourceAccountNotFound
		}
		if _, ok := balances[transfer.ToAccountID]; !ok {
			return entity.ErrDestAccountNotFound
		}
		if sourceBalance < transfer.Amount {
			return entity.ErrInsufficientFunds
		}

		_, err = r.db(ctx).Exec(ctx, `
			UPDATE users
			SET balance = balance - $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, transfer.Amount, transfer.FromAccountID)
		if err != nil {
			return fmt.Errorf("update source account: %w", err)
		}

		_, err = r.db(ctx).Exec(ctx, `
			UPDATE users
			SET balance = balance + $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, transfer.Amount, transfer.ToAccountID)
		if err != nil {
			return fmt.Errorf("update destination account: %w", err)
		}

		_, err = r.db(ctx).Exec(ctx, `
			INSERT INTO transactions(from_user_id, to_user_id, amount)
			VALUES($1, $2, $3)
		`, transfer.FromAccountID, transfer.ToAccountID, transfer.Amount)
		if err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		return nil
	})
}
