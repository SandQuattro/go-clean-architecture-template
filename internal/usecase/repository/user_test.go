package repository

import (
	"context"
	"testing"

	tx "github.com/Thiht/transactor/pgx"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"

	"clean-arch-template/internal/entity"

	"github.com/stretchr/testify/assert"
)

func TestUserRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Function to create a new mock database for each test
	newMockDB := func() (pgxmock.PgxConnIface, *UserRepository) {
		mockDb, err := pgxmock.NewConn()
		require.NoError(t, err)

		// Create mock DBGetter that returns our mockDb
		dbGetter := tx.DBGetter(func(ctx context.Context) tx.DB {
			return mockDb
		})

		// Create mock transactor
		transactor := &tx.Transactor{}

		repo := NewUserRepository(dbGetter, transactor)
		return mockDb, repo
	}

	t.Run("test InsertUser", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		var user entity.User
		user.Name = "test"

		mockDb.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))

		result, err := repo.InsertUser(ctx, &user)
		require.NoError(t, err)
		assert.Equal(t, 1, result.ID)
		assert.Equal(t, "test", result.Name)

		err = mockDb.ExpectationsWereMet()
		require.NoError(t, err)
	})

	t.Run("test UpdateUser", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		var user entity.User
		user.ID = 1
		user.Name = "test"

		mockDb.ExpectExec("UPDATE users").
			WithArgs(user.ID, user.Name).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		result, err := repo.UpdateUser(ctx, &user)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Name, result.Name)

		err = mockDb.ExpectationsWereMet()
		require.NoError(t, err)
	})

	t.Run("test DeleteUser", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		var user entity.User
		user.ID = 1

		mockDb.ExpectExec("DELETE FROM users").
			WithArgs(user.ID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.DeleteUser(ctx, &user)
		require.NoError(t, err)

		err = mockDb.ExpectationsWereMet()
		require.NoError(t, err)
	})

	t.Run("test GetUserByID", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		var user entity.User
		user.ID = 1
		user.Name = "test"

		rows := pgxmock.NewRows([]string{"id", "name"}).
			AddRow(user.ID, user.Name)

		mockDb.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(user.ID).
			WillReturnRows(rows)

		result, err := repo.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.ID)
		assert.Equal(t, user.Name, result.Name)

		err = mockDb.ExpectationsWereMet()
		require.NoError(t, err)
	})

	t.Run("test GetAllUsers", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

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
		assert.Equal(t, len(users), len(result))
		for i, u := range result {
			assert.Equal(t, users[i].ID, u.ID)
			assert.Equal(t, users[i].Name, u.Name)
		}

		err = mockDb.ExpectationsWereMet()
		require.NoError(t, err)
	})

	t.Run("test GetAllUsersWithOrders", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

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

		err = mockDb.ExpectationsWereMet()
		require.NoError(t, err)
	})

	t.Run("test GetAllUsersWithOrders with orders", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

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

		err = mockDb.ExpectationsWereMet()
		require.NoError(t, err)
	})
}
