package repository

import (
	"clean-arch-template/internal/entity"
	"context"
	"testing"

	tx "github.com/Thiht/transactor/pgx"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTransactor исполняет функцию без реальной транзакции: pgxmock проверяет
// сами запросы, а протокол Begin/Commit — зона ответственности библиотеки transactor.
type fakeTransactor struct{}

func (fakeTransactor) WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error {
	return txFunc(ctx)
}

func newMockDB(t *testing.T) (pgxmock.PgxConnIface, *UserRepository) {
	t.Helper()

	mockDb, err := pgxmock.NewConn()
	require.NoError(t, err)
	t.Cleanup(func() { _ = mockDb.Close(context.Background()) })

	dbGetter := tx.DBGetter(func(ctx context.Context) tx.DB {
		return mockDb
	})

	repo := NewUserRepository(dbGetter, fakeTransactor{})

	return mockDb, repo
}

func TestUserRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("test InsertUser", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		var user entity.User
		user.Name = "test"

		mockDb.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))

		result, err := repo.InsertUser(ctx, &user)
		require.NoError(t, err)
		assert.Equal(t, 1, result.ID)
		assert.Equal(t, "test", result.Name)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test UpdateUser", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		user := entity.User{ID: 1, Name: "test"}

		mockDb.ExpectQuery("UPDATE users").
			WithArgs(user.ID, user.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id", "name"}).AddRow(1, "test"))

		result, err := repo.UpdateUser(ctx, &user)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Name, result.Name)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test UpdateUser not found", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		mockDb.ExpectQuery("UPDATE users").
			WithArgs(999, "ghost").
			WillReturnError(pgx.ErrNoRows)

		result, err := repo.UpdateUser(ctx, &entity.User{ID: 999, Name: "ghost"})
		require.ErrorIs(t, err, entity.ErrUserNotFound)
		assert.Nil(t, result)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test DeleteUser", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		mockDb.ExpectExec("DELETE FROM users").
			WithArgs(1).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.DeleteUser(ctx, 1)
		require.NoError(t, err)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test DeleteUser not found", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		mockDb.ExpectExec("DELETE FROM users").
			WithArgs(999).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.DeleteUser(ctx, 999)
		require.ErrorIs(t, err, entity.ErrUserNotFound)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test GetUserByID", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		user := entity.User{ID: 1, Name: "test"}

		rows := pgxmock.NewRows([]string{"id", "name"}).
			AddRow(user.ID, user.Name)

		mockDb.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(user.ID).
			WillReturnRows(rows)

		result, err := repo.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Name, result.Name)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test GetUserByID not found", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		mockDb.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(999).
			WillReturnError(pgx.ErrNoRows)

		result, err := repo.GetUserByID(ctx, 999)
		require.ErrorIs(t, err, entity.ErrUserNotFound)
		assert.Nil(t, result)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test GetAllUsers", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		users := []entity.User{
			{ID: 1, Name: "test1"},
			{ID: 2, Name: "test2"},
		}

		rows := pgxmock.NewRows([]string{"id", "name"})
		for _, u := range users {
			rows.AddRow(u.ID, u.Name)
		}

		mockDb.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(0, 10).
			WillReturnRows(rows)

		result, err := repo.GetAllUsers(ctx, 0, 10)
		require.NoError(t, err)
		assert.Equal(t, users, result)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test GetAllUsersWithOrders", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		users := []entity.User{
			{ID: 1, Name: "test1"},
			{ID: 2, Name: "test2"},
		}

		rows := pgxmock.NewRows([]string{"id", "name", "order_ids", "order_amounts"})
		for _, u := range users {
			rows.AddRow(u.ID, u.Name, nil, nil)
		}

		mockDb.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(0, 10).
			WillReturnRows(rows)

		result, err := repo.GetAllUsersWithOrders(ctx, 0, 10)
		require.NoError(t, err)
		assert.Equal(t, len(users), len(result))
		for i, u := range result {
			assert.Equal(t, users[i].ID, u.ID)
			assert.Equal(t, users[i].Name, u.Name)
		}

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("test GetAllUsersWithOrders with orders", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		// Prepare rows: first user has orders, second user has no orders
		rows := pgxmock.NewRows([]string{"id", "name", "order_ids", "order_amounts"}).
			AddRow(1, "test1", []int64{10, 20}, []int64{100, 200}).
			AddRow(2, "test2", nil, nil)

		mockDb.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(0, 10).
			WillReturnRows(rows)

		result, err := repo.GetAllUsersWithOrders(context.Background(), 0, 10)
		require.NoError(t, err)
		require.Len(t, result, 2)

		// Assert results for first user with orders
		u1 := result[0]
		assert.Equal(t, 1, u1.ID)
		assert.Equal(t, "test1", u1.Name)
		require.Len(t, u1.Orders, 2)
		assert.Equal(t, int64(10), u1.Orders[0].ID)
		assert.Equal(t, int64(100), u1.Orders[0].Amount)
		assert.Equal(t, int64(20), u1.Orders[1].ID)
		assert.Equal(t, int64(200), u1.Orders[1].Amount)

		// Assert results for second user with no orders
		u2 := result[1]
		assert.Equal(t, 2, u2.ID)
		assert.Equal(t, "test2", u2.Name)
		assert.Empty(t, u2.Orders)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})
}

func TestTransferMoney(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	transfer := entity.Transfer{FromAccountID: 1, ToAccountID: 2, Amount: 300}

	lockRows := func(rows ...[]any) *pgxmock.Rows {
		r := pgxmock.NewRows([]string{"id", "balance"})
		for _, row := range rows {
			r.AddRow(row...)
		}
		return r
	}

	t.Run("successful transfer locks both accounts and writes transaction", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		mockDb.ExpectQuery("SELECT id, balance FROM users").
			WithArgs([]int64{1, 2}).
			WillReturnRows(lockRows([]any{int64(1), int64(1000)}, []any{int64(2), int64(500)}))
		mockDb.ExpectExec("UPDATE users").
			WithArgs(int64(300), int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mockDb.ExpectExec("UPDATE users").
			WithArgs(int64(300), int64(2)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mockDb.ExpectExec("INSERT INTO transactions").
			WithArgs(int64(1), int64(2), int64(300)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.TransferMoney(ctx, transfer)
		require.NoError(t, err)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("insufficient funds", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		mockDb.ExpectQuery("SELECT id, balance FROM users").
			WithArgs([]int64{1, 2}).
			WillReturnRows(lockRows([]any{int64(1), int64(100)}, []any{int64(2), int64(500)}))

		err := repo.TransferMoney(ctx, transfer)
		require.ErrorIs(t, err, entity.ErrInsufficientFunds)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("source account not found", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		mockDb.ExpectQuery("SELECT id, balance FROM users").
			WithArgs([]int64{1, 2}).
			WillReturnRows(lockRows([]any{int64(2), int64(500)}))

		err := repo.TransferMoney(ctx, transfer)
		require.ErrorIs(t, err, entity.ErrSourceAccountNotFound)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("destination account not found", func(t *testing.T) {
		mockDb, repo := newMockDB(t)

		mockDb.ExpectQuery("SELECT id, balance FROM users").
			WithArgs([]int64{1, 2}).
			WillReturnRows(lockRows([]any{int64(1), int64(1000)}))

		err := repo.TransferMoney(ctx, transfer)
		require.ErrorIs(t, err, entity.ErrDestAccountNotFound)

		require.NoError(t, mockDb.ExpectationsWereMet())
	})
}
