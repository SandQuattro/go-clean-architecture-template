package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
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

	t.Run("Empty Name", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		// Входные данные
		input := &entity.User{Name: ""}

		// Ожидаемое поведение
		mockDb.ExpectQuery("INSERT INTO users").
			WithArgs(input.Name).
			WillReturnError(ErrInvalidInputData)

		// Вызов функции
		result, err := repo.InsertUser(context.Background(), input)

		// Проверка результата
		assert.Equal(t, ErrInvalidInputData, err)
		assert.Nil(t, result)

		// Проверка, что не было вызовов к базе данных
		require.Error(t, mockDb.ExpectationsWereMet())
	})

	t.Run("Invalid UTF-8 Name", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		// Входные данные с некорректной кодировкой
		input := &entity.User{Name: string([]byte{0xbf, 0x27, 0x38})}

		// Ожидаемое поведение
		mockDb.ExpectQuery("INSERT INTO users").
			WithArgs(input.Name).
			WillReturnError(ErrInvalidInputData)

		// Вызов функции
		result, err := repo.InsertUser(context.Background(), input)

		// Проверка результата
		assert.Error(t, err)
		assert.Nil(t, result)

		// Проверка, что не было вызовов к базе данных
		require.Error(t, mockDb.ExpectationsWereMet())
	})

	t.Run("Valid Name", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		// Входные данные для вставки
		input := &entity.User{Name: "John"}

		// Ожидания для мока
		mockDb.ExpectQuery("INSERT").
			WithArgs(input.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))

		// Вызов функции
		result, e := repo.InsertUser(context.Background(), input)

		// Проверка результата
		require.NoError(t, e)
		require.NotNil(t, result)
		require.Equal(t, 1, result.ID)
		require.Equal(t, "John", result.Name)

		// Проверка, что не было вызовов к базе данных
		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("UpdateUser", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		user := entity.User{ID: 1, Name: "Updated Name"}

		mockDb.ExpectExec("UPDATE users").
			WithArgs(user.ID, user.Name).
			WillReturnResult(pgconn.NewCommandTag(fmt.Sprintf("%s %d", "UPDATE", 1)))

		updatedUser, err := repo.UpdateUser(ctx, &user)
		require.NoError(t, err)
		require.NotNil(t, updatedUser)

		// Проверка, что не было вызовов к базе данных
		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("GetAllUsers", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
			WithArgs(0, 10).
			WillReturnRows(pgxmock.NewRows([]string{"id", "name"}).AddRow(1, "John"))

		users, err := repo.GetAllUsers(ctx, 0, 10)
		require.NoError(t, err)
		require.NotNil(t, users)

		// Проверка, что не было вызовов к базе данных
		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	// TODO fix test
	t.Run("GetUserByID", func(t *testing.T) {
		id := 1
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		mockDb.ExpectQuery("SELECT u.id, u.name, o.id as order_id, o.amount as order_amount FROM users").
			WithArgs(id).
			WillReturnRows(pgxmock.NewRows([]string{"id", "name"}).AddRow(1, "John"))

		user, err := repo.GetUserByID(ctx, id)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, user.ID, id)

		// Проверка, что не было вызовов к базе данных
		require.NoError(t, mockDb.ExpectationsWereMet())
	})

	t.Run("DeleteUser", func(t *testing.T) {
		mockDb, repo := newMockDB()
		defer mockDb.Close(context.Background())

		userToDelete := entity.User{Name: "To Be Deleted"}

		mockDb.ExpectExec("DELETE FROM users").
			WithArgs(userToDelete.ID).
			WillReturnResult(pgconn.NewCommandTag(fmt.Sprintf("%s %d", "DELETE", 1)))

		err := repo.DeleteUser(ctx, &userToDelete)
		require.NoError(t, err)

		// Проверка, что не было вызовов к базе данных
		require.NoError(t, mockDb.ExpectationsWereMet())
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
		user.ID = 0 // Ensure ID is 0 for insertion
		user.Name = name

		db.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))

		_, err = repo.InsertUser(ctx, &user)
		require.NoError(t, err)
		// assert.NotNil(t, insertedUser)
		// require.NotEqual(t, 0, insertedUser.ID)
		// require.Equal(t, user.Name, insertedUser.Name)
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
