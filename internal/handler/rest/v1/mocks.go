// Code generated by MockGen. DO NOT EDIT.
// Source: interfaces.go

// Package v1 is a generated GoMock package.
package v1

import (
	entity "clean-arch-template/internal/entity"
	usecase "clean-arch-template/internal/usecase"
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockUserUseCase is a mock of UserUseCase interface.
type MockUserUseCase struct {
	ctrl     *gomock.Controller
	recorder *MockUserUseCaseMockRecorder
}

// MockUserUseCaseMockRecorder is the mock recorder for MockUserUseCase.
type MockUserUseCaseMockRecorder struct {
	mock *MockUserUseCase
}

// NewMockUserUseCase creates a new mock instance.
func NewMockUserUseCase(ctrl *gomock.Controller) *MockUserUseCase {
	mock := &MockUserUseCase{ctrl: ctrl}
	mock.recorder = &MockUserUseCaseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserUseCase) EXPECT() *MockUserUseCaseMockRecorder {
	return m.recorder
}

// CreateUser mocks base method.
func (m *MockUserUseCase) CreateUser(ctx context.Context, cmd usecase.CreateUpdateDeleteUserCommand) (*entity.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, cmd)
	ret0, _ := ret[0].(*entity.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockUserUseCaseMockRecorder) CreateUser(ctx, cmd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockUserUseCase)(nil).CreateUser), ctx, cmd)
}

// DeleteUser mocks base method.
func (m *MockUserUseCase) DeleteUser(ctx context.Context, cmd usecase.CreateUpdateDeleteUserCommand) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUser", ctx, cmd)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockUserUseCaseMockRecorder) DeleteUser(ctx, cmd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockUserUseCase)(nil).DeleteUser), ctx, cmd)
}

// FindAllUsers mocks base method.
func (m *MockUserUseCase) FindAllUsers(ctx context.Context, cmd usecase.FindAllUsersCommand) ([]entity.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindAllUsers", ctx, cmd)
	ret0, _ := ret[0].([]entity.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindAllUsers indicates an expected call of FindAllUsers.
func (mr *MockUserUseCaseMockRecorder) FindAllUsers(ctx, cmd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindAllUsers", reflect.TypeOf((*MockUserUseCase)(nil).FindAllUsers), ctx, cmd)
}

// FindUserByID mocks base method.
func (m *MockUserUseCase) FindUserByID(ctx context.Context, cmd usecase.FindUserByIDCommand) (*entity.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindUserByID", ctx, cmd)
	ret0, _ := ret[0].(*entity.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindUserByID indicates an expected call of FindUserByID.
func (mr *MockUserUseCaseMockRecorder) FindUserByID(ctx, cmd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindUserByID", reflect.TypeOf((*MockUserUseCase)(nil).FindUserByID), ctx, cmd)
}

// UpdateUser mocks base method.
func (m *MockUserUseCase) UpdateUser(ctx context.Context, cmd usecase.CreateUpdateDeleteUserCommand) (*entity.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", ctx, cmd)
	ret0, _ := ret[0].(*entity.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockUserUseCaseMockRecorder) UpdateUser(ctx, cmd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockUserUseCase)(nil).UpdateUser), ctx, cmd)
}
