package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"

	"clean-arch-template/internal/entity"

	"github.com/stretchr/testify/assert"
)

func TestUserRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Функция для создания новой мока базы данных для каждого теста
	newMockDB := func() (pgxmock.PgxConnIface, *UserRepository) {
		mockDb, err := pgxmock.NewConn()
		require.NoError(t, err)
		repo := &UserRepository{db: mockDb}
		return mockDb, repo
	}

	t.Run("InsertUser", func(t *testing.T) {
		t.Run("Empty Name", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			input := &entity.User{Name: ""}

			result, err := repo.InsertUser(context.Background(), input)

			assert.Equal(t, ErrInvalidInputData, err)
			assert.Nil(t, result)
		})

		t.Run("Invalid UTF-8 Name", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			input := &entity.User{Name: string([]byte{0xbf, 0x27, 0x38})}

			result, err := repo.InsertUser(context.Background(), input)

			assert.Error(t, err)
			assert.Nil(t, result)
		})

		t.Run("Valid Name", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			input := &entity.User{Name: "John"}

			mockDb.ExpectQuery("INSERT INTO users").
				WithArgs(input.Name).
				WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))

			result, err := repo.InsertUser(context.Background(), input)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, 1, result.ID)
			require.Equal(t, "John", result.Name)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("Database Error", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			input := &entity.User{Name: "John"}

			mockDb.ExpectQuery("INSERT INTO users").
				WithArgs(input.Name).
				WillReturnError(fmt.Errorf("database connection error"))

			result, err := repo.InsertUser(context.Background(), input)

			require.Error(t, err)
			require.Nil(t, result)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})
	})

	t.Run("UpdateUser", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			user := entity.User{ID: 1, Name: "Updated Name"}

			mockDb.ExpectExec("UPDATE users").
				WithArgs(user.ID, user.Name).
				WillReturnResult(pgconn.NewCommandTag("UPDATE 1"))

			updatedUser, err := repo.UpdateUser(ctx, &user)
			require.NoError(t, err)
			require.NotNil(t, updatedUser)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("No Rows Updated", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			user := entity.User{ID: 999, Name: "Non Existent"}

			mockDb.ExpectExec("UPDATE users").
				WithArgs(user.ID, user.Name).
				WillReturnResult(pgconn.NewCommandTag("UPDATE 0"))

			updatedUser, err := repo.UpdateUser(ctx, &user)
			require.NoError(t, err) // В текущей реализации нет проверки на количество обновленных строк
			require.NotNil(t, updatedUser)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("Database Error", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			user := entity.User{ID: 1, Name: "Error Test"}

			mockDb.ExpectExec("UPDATE users").
				WithArgs(user.ID, user.Name).
				WillReturnError(fmt.Errorf("database connection error"))

			updatedUser, err := repo.UpdateUser(ctx, &user)
			require.Error(t, err)
			require.Nil(t, updatedUser)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})
	})

	t.Run("GetAllUsers", func(t *testing.T) {
		t.Run("Success With Orders", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			rows := pgxmock.NewRows([]string{"id", "name", "order_id", "order_amount"}).
				AddRow(1, "John", pgtype.Int4{Int32: 1, Valid: true}, pgtype.Int4{Int32: 100, Valid: true}).
				AddRow(1, "John", pgtype.Int4{Int32: 2, Valid: true}, pgtype.Int4{Int32: 200, Valid: true}).
				AddRow(2, "Jane", pgtype.Int4{Int32: 3, Valid: true}, pgtype.Int4{Int32: 300, Valid: true})

			mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
				WithArgs(0, 10).
				WillReturnRows(rows)

			users, err := repo.GetAllUsers(ctx, 0, 10)
			require.NoError(t, err)
			require.Len(t, users, 2) // Должно быть 2 пользователя
			require.Equal(t, "John", users[0].Name)
			require.Len(t, users[0].Orders, 2) // У John должно быть 2 заказа
			require.Equal(t, "Jane", users[1].Name)
			require.Len(t, users[1].Orders, 1) // У Jane должен быть 1 заказ
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("Empty Result", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
				WithArgs(0, 10).
				WillReturnRows(pgxmock.NewRows([]string{"id", "name", "order_id", "order_amount"}))

			users, err := repo.GetAllUsers(ctx, 0, 10)
			require.NoError(t, err)
			require.Empty(t, users)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("Database Error", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
				WithArgs(0, 10).
				WillReturnError(fmt.Errorf("database connection error"))

			users, err := repo.GetAllUsers(ctx, 0, 10)
			require.Error(t, err)
			require.Nil(t, users)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("Invalid Pagination", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
				WithArgs(-1, -10).
				WillReturnError(fmt.Errorf("invalid pagination parameters"))

			users, err := repo.GetAllUsers(ctx, -1, -10)
			require.Error(t, err)
			require.Nil(t, users)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})
	})

	t.Run("GetUserByID", func(t *testing.T) {
		t.Run("Success With Orders", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			rows := pgxmock.NewRows([]string{"id", "name", "order_id", "order_amount"}).
				AddRow(1, "John", pgtype.Int4{Int32: 1, Valid: true}, pgtype.Int4{Int32: 100, Valid: true}).
				AddRow(1, "John", pgtype.Int4{Int32: 2, Valid: true}, pgtype.Int4{Int32: 200, Valid: true})

			mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
				WithArgs(1).
				WillReturnRows(rows)

			user, err := repo.GetUserByID(ctx, 1)
			require.NoError(t, err)
			require.NotNil(t, user)
			require.Equal(t, 1, user.ID)
			require.Equal(t, "John", user.Name)
			require.Len(t, user.Orders, 2)
			require.Equal(t, int64(100), user.Orders[0].Amount)
			require.Equal(t, int64(200), user.Orders[1].Amount)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("No Orders", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			rows := pgxmock.NewRows([]string{"id", "name", "order_id", "order_amount"}).
				AddRow(1, "John", pgtype.Int4{Valid: false}, pgtype.Int4{Valid: false})

			mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
				WithArgs(1).
				WillReturnRows(rows)

			user, err := repo.GetUserByID(ctx, 1)
			require.NoError(t, err)
			require.NotNil(t, user)
			require.Equal(t, 1, user.ID)
			require.Equal(t, "John", user.Name)
			require.Empty(t, user.Orders)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("User Not Found", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
				WithArgs(999).
				WillReturnRows(pgxmock.NewRows([]string{"id", "name", "order_id", "order_amount"}))

			user, err := repo.GetUserByID(ctx, 999)
			require.NoError(t, err) // В текущей реализации нет проверки на отсутствие пользователя
			require.Nil(t, user)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("Database Error", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
				WithArgs(1).
				WillReturnError(fmt.Errorf("database connection error"))

			user, err := repo.GetUserByID(ctx, 1)
			require.Error(t, err)
			require.Nil(t, user)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})
	})

	t.Run("DeleteUser", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			userToDelete := entity.User{ID: 1, Name: "To Be Deleted"}

			mockDb.ExpectExec("DELETE FROM users").
				WithArgs(userToDelete.ID).
				WillReturnResult(pgconn.NewCommandTag("DELETE 1"))

			err := repo.DeleteUser(ctx, &userToDelete)
			require.NoError(t, err)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("No Rows Deleted", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			userToDelete := entity.User{ID: 999, Name: "Non Existent"}

			mockDb.ExpectExec("DELETE FROM users").
				WithArgs(userToDelete.ID).
				WillReturnResult(pgconn.NewCommandTag("DELETE 0"))

			err := repo.DeleteUser(ctx, &userToDelete)
			require.NoError(t, err) // В текущей реализации нет проверки на количество удаленных строк
			require.NoError(t, mockDb.ExpectationsWereMet())
		})

		t.Run("Database Error", func(t *testing.T) {
			mockDb, repo := newMockDB()
			defer mockDb.Close(context.Background())

			userToDelete := entity.User{ID: 1, Name: "Error Test"}

			mockDb.ExpectExec("DELETE FROM users").
				WithArgs(userToDelete.ID).
				WillReturnError(fmt.Errorf("database connection error"))

			err := repo.DeleteUser(ctx, &userToDelete)
			require.Error(t, err)
			require.NoError(t, mockDb.ExpectationsWereMet())
		})
	})
}

func FuzzUserRepository(f *testing.F) {
	ctx := context.Background()

	// Создаём мокируемую базу данных
	db, err := pgxmock.NewConn()
	if err != nil {
		f.Fatalf("failed to create mock connection: %v", err)
	}
	defer db.Close(context.Background())

	o := sync.Once{}
	repo := NewUserRepository(&o, db)

	f.Fuzz(func(t *testing.T, name string) {
		var user entity.User
		user.ID = 0 // ID = 0 for insertion
		user.Name = name

		db.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))

		_, err = repo.InsertUser(ctx, &user)
		require.NoError(t, err)
	})

	//t.Run("FuzzyUpdateUser", func(t *testing.T) {
	//	f := fuzz.New()
	//	var user entity.User
	//	for i := 0; i < 100; i++ {
	//		f.Fuzz(&user)
	//		user.ID = i + 1 // Ensure a valid ID for update
	//		updatedUser, err := repo.UpdateUser(ctx, &user)
	//		assert.NoError(t, err)
	//		assert.NotNil(t, updatedUser)
	//		assert.Equal(t, user.ID, updatedUser.ID)
	//		assert.Equal(t, user.Name, updatedUser.Name)
	//	}
	//})
	//
	//t.Run("FuzzyGetUserByID", func(t *testing.T) {
	//	f := fuzz.New()
	//	var id int
	//	for i := 0; i < 100; i++ {
	//		f.Fuzz(&id)
	//		user, err := repo.GetUserByID(ctx, id)
	//		if err != nil {
	//			assert.Nil(t, user)
	//		} else {
	//			assert.NotNil(t, user)
	//			assert.Equal(t, id, user.ID)
	//		}
	//	}
	//})
	//
	//t.Run("FuzzyDeleteUser", func(t *testing.T) {
	//	f := fuzz.New()
	//	var user entity.User
	//	for i := 0; i < 100; i++ {
	//		f.Fuzz(&user)
	//		user.ID = i + 1 // Ensure a valid ID for deletion
	//		err := repo.DeleteUser(ctx, &user)
	//		assert.NoError(t, err)
	//
	//		deletedUser, err := repo.GetUserByID(ctx, user.ID)
	//		assert.Error(t, err)
	//		assert.Nil(t, deletedUser)
	//	}
	//})
	// Проверка на то, что все ожидания были выполнены
	if err = db.ExpectationsWereMet(); err != nil {
		f.Errorf("there were unfulfilled expectations: %v", err)
	}
}
