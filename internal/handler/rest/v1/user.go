package v1

import (
	"context"

	"clean-arch-template/internal/entity"
	"clean-arch-template/internal/usecase"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/danielgtaylor/huma/v2"
)

var _ UserUseCase = (*usecase.UserUseCase)(nil)

const tracerName = "user handler"

type UserHandler struct {
	userUC UserUseCase
}

func NewUserHandler(uc UserUseCase) *UserHandler {
	return &UserHandler{userUC: uc}
}

func (uh *UserHandler) ListUsers(ctx context.Context, req *ListUserRequest) (*ListUserResponse, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "ListUsers", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// Validate input parameters
	if req.Page < 0 || req.Size <= 0 {
		return nil, huma.Error400BadRequest("invalid pagination parameters: page must be equal or greater then 0 and size must be greater then 0")
	}

	cmd := usecase.FindAllUsersCommand{
		Page: req.Page,
		Size: req.Size,
	}

	users, err := uh.userUC.FindAllUsers(ctx, cmd)
	if err != nil {
		return nil, huma.Error500InternalServerError("error fetching users: ", err)
	}

	return &ListUserResponse{Body: struct{ Users []entity.User }{Users: users}}, nil
}

func (uh *UserHandler) FindUserByID(ctx context.Context, req *FindUserRequest) (*UserResponse, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "FindUserByID", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	cmd := usecase.FindUserByIDCommand{ID: req.ID}

	user, err := uh.userUC.FindUserByID(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}
	if user == nil {
		return nil, MapError(usecase.ErrUserNotFound)
	}

	resp := &UserResponse{
		Body: struct{ *entity.User }{User: user},
	}

	return resp, nil
}

func (uh *UserHandler) CreateUser(ctx context.Context, req *UserRequest) (*UserResponse, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "CreateUser", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if req.Body.User.Name == "" {
		return nil, MapError(ErrEmptyName)
	}

	cmd := usecase.CreateUpdateUserCommand{User: req.Body.User}

	user, err := uh.userUC.CreateUser(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}

	resp := &UserResponse{
		Body: struct{ *entity.User }{User: user},
	}

	return resp, nil
}

func (uh *UserHandler) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UserResponse, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "UpdateUser", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if req.Body.User.Name == "" {
		return nil, MapError(ErrEmptyName)
	}

	cmd := usecase.CreateUpdateUserCommand{User: req.Body.User}

	user, err := uh.userUC.UpdateUser(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}

	resp := &UserResponse{
		Body: struct{ *entity.User }{User: user},
	}

	return resp, nil
}

func (uh *UserHandler) DeleteUser(ctx context.Context, req *FindUserRequest) (*struct{}, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "DeleteUser", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	cmd := usecase.DeleteUserByIDCommand{ID: req.ID}

	err := uh.userUC.DeleteUser(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}

	return &struct{}{}, nil
}

func (uh *UserHandler) TransferMoney(ctx context.Context, req *TransferMoneyRequest) (*struct{}, error) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "TransferMoney", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	cmd := usecase.TransferMoneyCommand{Transfer: *req.Body.Transfer}

	err := uh.userUC.TransferMoney(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}

	return &struct{}{}, nil
}
