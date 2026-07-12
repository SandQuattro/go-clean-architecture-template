package v1

import (
	"clean-arch-template/internal/usecase"
	"context"

	"go.opentelemetry.io/otel"
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
	// otelfiber уже создаёт server-спан; здесь — вложенный internal-спан,
	// новый ctx передаётся вниз, чтобы спаны usecase/pgx стали его детьми.
	ctx, span := otel.Tracer(tracerName).Start(ctx, "ListUsers")
	defer span.End()

	cmd := usecase.FindAllUsersCommand{
		Page: req.Page,
		Size: req.Size,
	}

	users, err := uh.userUC.FindAllUsers(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}

	return ToUserListOutputFromEntity(users), nil
}

func (uh *UserHandler) FindUserByID(ctx context.Context, req *FindUserRequest) (*UserResponse, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "FindUserByID")
	defer span.End()

	cmd := usecase.FindUserByIDCommand{ID: req.ID}

	user, err := uh.userUC.FindUserByID(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}

	return ToUserOutputFromEntity(user), nil
}

func (uh *UserHandler) CreateUser(ctx context.Context, req *CreateUserRequest) (*UserResponse, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "CreateUser")
	defer span.End()

	cmd := usecase.CreateUpdateUserCommand{}
	cmd.User.Name = req.Body.Name

	user, err := uh.userUC.CreateUser(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}

	return ToUserOutputFromEntity(user), nil
}

func (uh *UserHandler) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UserResponse, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "UpdateUser")
	defer span.End()

	// ID берётся только из пути: ID в теле игнорируется, чтобы PUT /user/5
	// не мог обновить чужую запись.
	cmd := usecase.CreateUpdateUserCommand{}
	cmd.User.ID = req.ID
	cmd.User.Name = req.Body.Name

	user, err := uh.userUC.UpdateUser(ctx, cmd)
	if err != nil {
		return nil, MapError(err)
	}

	return ToUserOutputFromEntity(user), nil
}

func (uh *UserHandler) DeleteUser(ctx context.Context, req *FindUserRequest) (*struct{}, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "DeleteUser")
	defer span.End()

	cmd := usecase.DeleteUserByIDCommand{ID: req.ID}

	if err := uh.userUC.DeleteUser(ctx, cmd); err != nil {
		return nil, MapError(err)
	}

	return &struct{}{}, nil
}

func (uh *UserHandler) TransferMoney(ctx context.Context, req *TransferMoneyRequest) (*struct{}, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "TransferMoney")
	defer span.End()

	cmd := usecase.TransferMoneyCommand{Transfer: ToTransferEntity(req.Body)}

	if err := uh.userUC.TransferMoney(ctx, cmd); err != nil {
		return nil, MapError(err)
	}

	return &struct{}{}, nil
}
