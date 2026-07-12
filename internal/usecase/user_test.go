package usecase

import (
	"clean-arch-template/internal/entity"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errInternalServErr = errors.New("internal server error")

func newUseCase(t *testing.T) (*UserUseCase, *MockUserRepository) {
	t.Helper()

	repo := NewMockUserRepository(gomock.NewController(t))

	return NewUserUseCase(repo), repo
}

func TestFindAllUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  FindAllUsersCommand
		mock func(repo *MockUserRepository)
		res  []entity.User
		err  error
	}{
		{
			name: "first page maps to zero offset",
			cmd:  FindAllUsersCommand{Page: 1, Size: 10},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().GetAllUsers(gomock.Any(), 0, 10).Return([]entity.User{{ID: 1, Name: "A"}}, nil)
			},
			res: []entity.User{{ID: 1, Name: "A"}},
		},
		{
			name: "second page starts right after the first",
			cmd:  FindAllUsersCommand{Page: 2, Size: 10},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().GetAllUsers(gomock.Any(), 10, 10).Return([]entity.User{}, nil)
			},
			res: []entity.User{},
		},
		{
			name: "zero page is rejected",
			cmd:  FindAllUsersCommand{Page: 0, Size: 10},
			mock: func(repo *MockUserRepository) {},
			err:  entity.ErrInvalidPagination,
		},
		{
			name: "zero size is rejected",
			cmd:  FindAllUsersCommand{Page: 1, Size: 0},
			mock: func(repo *MockUserRepository) {},
			err:  entity.ErrInvalidPagination,
		},
		{
			name: "repository error is propagated",
			cmd:  FindAllUsersCommand{Page: 1, Size: 10},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().GetAllUsers(gomock.Any(), 0, 10).Return(nil, errInternalServErr)
			},
			err: errInternalServErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userUseCase, repo := newUseCase(t)
			tc.mock(repo)

			res, err := userUseCase.FindAllUsers(context.Background(), tc.cmd)

			require.Equal(t, tc.res, res)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestFindUserByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   int
		mock func(repo *MockUserRepository)
		res  *entity.User
		err  error
	}{
		{
			name: "user found",
			id:   1,
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().GetUserByID(gomock.Any(), 1).Return(&entity.User{ID: 1, Name: "John Doe"}, nil)
			},
			res: &entity.User{ID: 1, Name: "John Doe"},
		},
		{
			name: "user not found",
			id:   2,
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().GetUserByID(gomock.Any(), 2).Return(nil, entity.ErrUserNotFound)
			},
			err: entity.ErrUserNotFound,
		},
		{
			name: "internal server error",
			id:   3,
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().GetUserByID(gomock.Any(), 3).Return(nil, errInternalServErr)
			},
			err: errInternalServErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userUseCase, repo := newUseCase(t)
			tc.mock(repo)

			res, err := userUseCase.FindUserByID(context.Background(), FindUserByIDCommand{ID: tc.id})

			require.Equal(t, tc.res, res)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     entity.User
		mock     func(repo *MockUserRepository)
		expected *entity.User
		err      error
	}{
		{
			name: "create user success",
			user: entity.User{Name: "Jane Doe"},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().InsertUser(gomock.Any(), &entity.User{Name: "Jane Doe"}).Return(&entity.User{ID: 1, Name: "Jane Doe"}, nil)
			},
			expected: &entity.User{ID: 1, Name: "Jane Doe"},
		},
		{
			name: "empty name is rejected without repository call",
			user: entity.User{Name: ""},
			mock: func(repo *MockUserRepository) {},
			err:  entity.ErrInvalidUserName,
		},
		{
			name: "invalid utf-8 name is rejected",
			user: entity.User{Name: string([]byte{0xff, 0xfe})},
			mock: func(repo *MockUserRepository) {},
			err:  entity.ErrInvalidUserName,
		},
		{
			name: "create user failure",
			user: entity.User{Name: "John Smith"},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().InsertUser(gomock.Any(), &entity.User{Name: "John Smith"}).Return(nil, errInternalServErr)
			},
			err: errInternalServErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userUseCase, repo := newUseCase(t)
			tc.mock(repo)

			result, err := userUseCase.CreateUser(context.Background(), CreateUpdateUserCommand{User: tc.user})

			require.Equal(t, tc.expected, result)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     entity.User
		mock     func(repo *MockUserRepository)
		expected *entity.User
		err      error
	}{
		{
			name: "update user success",
			user: entity.User{ID: 1, Name: "Jane Doe"},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().UpdateUser(gomock.Any(), &entity.User{ID: 1, Name: "Jane Doe"}).Return(&entity.User{ID: 1, Name: "Jane Doe"}, nil)
			},
			expected: &entity.User{ID: 1, Name: "Jane Doe"},
		},
		{
			name: "missing user surfaces not found",
			user: entity.User{ID: 999, Name: "Ghost"},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().UpdateUser(gomock.Any(), &entity.User{ID: 999, Name: "Ghost"}).Return(nil, entity.ErrUserNotFound)
			},
			err: entity.ErrUserNotFound,
		},
		{
			name: "empty name is rejected without repository call",
			user: entity.User{ID: 1, Name: ""},
			mock: func(repo *MockUserRepository) {},
			err:  entity.ErrInvalidUserName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userUseCase, repo := newUseCase(t)
			tc.mock(repo)

			result, err := userUseCase.UpdateUser(context.Background(), CreateUpdateUserCommand{User: tc.user})

			require.Equal(t, tc.expected, result)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   int
		mock func(repo *MockUserRepository)
		err  error
	}{
		{
			name: "delete user success",
			id:   1,
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().DeleteUser(gomock.Any(), 1).Return(nil)
			},
		},
		{
			name: "missing user surfaces not found",
			id:   999,
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().DeleteUser(gomock.Any(), 999).Return(entity.ErrUserNotFound)
			},
			err: entity.ErrUserNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userUseCase, repo := newUseCase(t)
			tc.mock(repo)

			err := userUseCase.DeleteUser(context.Background(), DeleteUserByIDCommand{ID: tc.id})

			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestTransferMoney(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		transfer entity.Transfer
		mock     func(repo *MockUserRepository)
		err      error
	}{
		{
			name:     "transfer success",
			transfer: entity.Transfer{FromAccountID: 1, ToAccountID: 2, Amount: 100},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().
					TransferMoney(gomock.Any(), entity.Transfer{FromAccountID: 1, ToAccountID: 2, Amount: 100}).
					Return(nil)
			},
		},
		{
			name:     "zero amount is rejected without repository call",
			transfer: entity.Transfer{FromAccountID: 1, ToAccountID: 2, Amount: 0},
			mock:     func(repo *MockUserRepository) {},
			err:      entity.ErrNegativeAmount,
		},
		{
			name:     "negative amount is rejected without repository call",
			transfer: entity.Transfer{FromAccountID: 1, ToAccountID: 2, Amount: -5},
			mock:     func(repo *MockUserRepository) {},
			err:      entity.ErrNegativeAmount,
		},
		{
			name:     "same account is rejected without repository call",
			transfer: entity.Transfer{FromAccountID: 1, ToAccountID: 1, Amount: 100},
			mock:     func(repo *MockUserRepository) {},
			err:      entity.ErrSameAccount,
		},
		{
			name:     "repository error is propagated",
			transfer: entity.Transfer{FromAccountID: 1, ToAccountID: 2, Amount: 100},
			mock: func(repo *MockUserRepository) {
				repo.EXPECT().
					TransferMoney(gomock.Any(), entity.Transfer{FromAccountID: 1, ToAccountID: 2, Amount: 100}).
					Return(entity.ErrInsufficientFunds)
			},
			err: entity.ErrInsufficientFunds,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userUseCase, repo := newUseCase(t)
			tc.mock(repo)

			err := userUseCase.TransferMoney(context.Background(), TransferMoneyCommand{Transfer: tc.transfer})

			require.ErrorIs(t, err, tc.err)
		})
	}
}
