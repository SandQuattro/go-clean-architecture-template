package repository

import (
	"context"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"

	"clean-arch-template/internal/entity"

	"github.com/stretchr/testify/assert"
)

func TestUserRepository(t *testing.T) {
	t.Parallel()
	// Setup
	ctx := context.Background()

	// Создаём мокируемую базу данных
	db, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("failed to create mock connection: %v", err)
	}
	defer db.Close(context.Background())

	// Создаём репозиторий с мокированной базой данных
	repo := &UserRepository{db: db}

	t.Run("Empty Name", func(t *testing.T) {
		// Входные данные
		input := &entity.User{Name: ""}

		// Вызов функции
		result, err := repo.InsertUser(context.Background(), input)

		// Проверка результата
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidInputData, err)
	})

	t.Run("Invalid UTF-8 Name", func(t *testing.T) {
		// Входные данные с некорректной кодировкой
		input := &entity.User{Name: string([]byte{0xbf, 0x27, 0x38})}

		// Ожидаемое поведение
		db.ExpectQuery("INSERT INTO users").
			WithArgs(input.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(1))

		// Вызов функции
		result, err := repo.InsertUser(context.Background(), input)

		// Проверка результата
		assert.Nil(t, result)
		assert.Error(t, err)
		//assert.Equal(t, 1, result.ID)
	})

	t.Run("Valid Name", func(t *testing.T) {
		// Входные данные с корректной кодировкой
		input := &entity.User{Name: "ValidName"}

		// Ожидаемое поведение
		db.ExpectQuery("INSERT INTO users").
			WithArgs(input.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(2))

		// Вызов функции
		result, err := repo.InsertUser(context.Background(), input)

		// Проверка результата
		require.NotNil(t, result)
		require.NoError(t, err)
		require.Equal(t, 2, result.ID)
	})

	// Проверка на то, что все ожидания были выполнены
	if err = db.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %v", err)
	}

	t.Run("UpdateUser", func(t *testing.T) {
		user := entity.User{ID: 1, Name: "Updated Name"}

		// Ожидаемое поведение
		db.ExpectQuery("UPDATE users").
			WithArgs(user.Name).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(2))

		_, err = repo.UpdateUser(ctx, &user)
		require.NoError(t, err)

		// Ожидаемое поведение
		db.ExpectQuery("SELECT * FROM users").
			WithArgs(user.ID).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(2))

		updatedUser, err := repo.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, updatedUser)
		require.Equal(t, "Updated Name", updatedUser.Name)
	})

	t.Run("GetAllUsers", func(t *testing.T) {
		users, err := repo.GetAllUsers(ctx, 0, 10)
		require.NoError(t, err)
		require.NotNil(t, users)
	})

	t.Run("GetUserByID", func(t *testing.T) {
		user, err := repo.GetUserByID(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, user)
		require.Equal(t, 1, user.ID)
	})

	t.Run("DeleteUser", func(t *testing.T) {
		userToDelete := entity.User{Name: "To Be Deleted"}
		insertedUser, err := repo.InsertUser(ctx, &userToDelete)
		require.NoError(t, err)

		err = repo.DeleteUser(ctx, insertedUser)
		require.NoError(t, err)

		deletedUser, err := repo.GetUserByID(ctx, userToDelete.ID)
		require.Nil(t, err) // Expecting an error as the user should not exist
		require.Nil(t, deletedUser)
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
		//assert.NotNil(t, insertedUser)
		//require.NotEqual(t, 0, insertedUser.ID)
		//require.Equal(t, user.Name, insertedUser.Name)
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
