package usecase

import (
	"context"
	"errors"
	"testing"

	"clean-arch-template/internal/entity"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	errInternalServErr = errors.New("internal server error")
	errNotFound        = errors.New("user not found")
)

type test struct {
	name string
	mock func()
	res  interface{}
	err  error
}

func user(t *testing.T) (*UserUseCase, *MockUserRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := NewMockUserRepository(mockCtl)

	userUseCase := NewUserUseCase(repo)
	return userUseCase, repo
}

func TestGetAllUsers(t *testing.T) {
	t.Parallel()

	userUseCase, userRepo := user(t)

	tests := []test{
		{
			name: "empty result",
			mock: func() {
				userRepo.EXPECT().GetAllUsers(context.Background(), 0, 10).Return([]entity.User{}, nil)
			},
			res: []entity.User{},
			err: nil,
		},
		{
			name: "result with error",
			mock: func() {
				userRepo.EXPECT().GetAllUsers(context.Background(), 0, 10).Return([]entity.User(nil), errInternalServErr)
			},
			res: []entity.User(nil),
			err: errInternalServErr,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.mock()

			res, err := userUseCase.FindAllUsers(
				context.Background(),
				FindAllUsersCommand{Page: 0, Size: 10})

			require.Equal(t, res, tc.res)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestFindUserByID(t *testing.T) {
	t.Parallel()

	userUseCase, userRepo := user(t)

	tests := []struct {
		name string
		id   int
		mock func()
		res  *entity.User
		err  error
	}{
		{
			name: "user found",
			id:   1,
			mock: func() {
				userRepo.EXPECT().GetUserByID(context.Background(), 1).Return(&entity.User{ID: 1, Name: "John Doe"}, nil)
			},
			res: &entity.User{ID: 1, Name: "John Doe"},
			err: nil,
		},
		{
			name: "user not found",
			id:   2,
			mock: func() {
				userRepo.EXPECT().GetUserByID(context.Background(), 2).Return(nil, errNotFound)
			},
			res: nil,
			err: errNotFound,
		},
		{
			name: "internal server error",
			id:   3,
			mock: func() {
				userRepo.EXPECT().GetUserByID(context.Background(), 3).Return(nil, errInternalServErr)
			},
			res: nil,
			err: errInternalServErr,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.mock()

			res, err := userUseCase.FindUserByID(
				context.Background(),
				FindUserByIDCommand{ID: tc.id})

			require.Equal(t, tc.res, res)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	userUseCase, userRepo := user(t)

	tests := []struct {
		name     string
		user     entity.User
		mock     func()
		expected *entity.User
		err      error
	}{
		{
			name: "create user success",
			user: entity.User{Name: "Jane Doe"},
			mock: func() {
				userRepo.EXPECT().InsertUser(context.Background(), &entity.User{Name: "Jane Doe"}).Return(&entity.User{ID: 1, Name: "Jane Doe"}, nil)
			},
			expected: &entity.User{ID: 1, Name: "Jane Doe"},
			err:      nil,
		},
		{
			name: "create user failure",
			user: entity.User{Name: "John Smith"},
			mock: func() {
				userRepo.EXPECT().InsertUser(context.Background(), &entity.User{Name: "John Smith"}).Return(nil, errInternalServErr)
			},
			expected: nil,
			err:      errInternalServErr,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.mock()

			result, err := userUseCase.CreateUser(
				context.Background(),
				CreateUpdateDeleteUserCommand{User: tc.user})

			require.Equal(t, tc.expected, result)
			require.ErrorIs(t, err, tc.err)
		})
	}
}
