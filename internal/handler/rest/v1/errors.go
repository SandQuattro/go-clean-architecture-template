package v1

import (
	"clean-arch-template/internal/entity"
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
)

// mapError маппит доменные ошибки в HTTP-ошибки. Неизвестные ошибки логируются
// (с trace_id из ctx) и уходят клиенту как generic 500.
func (uh *UserHandler) mapError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUserNotFound),
		errors.Is(err, entity.ErrSourceAccountNotFound),
		errors.Is(err, entity.ErrDestAccountNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, entity.ErrInvalidUserName),
		errors.Is(err, entity.ErrInvalidPagination),
		errors.Is(err, entity.ErrNegativeAmount),
		errors.Is(err, entity.ErrSameAccount):
		return huma.Error400BadRequest(err.Error())
	case errors.Is(err, entity.ErrInsufficientFunds):
		return huma.Error409Conflict(err.Error())
	default:
		uh.log.Error(ctx, "request failed", "error", err.Error())
		return huma.Error500InternalServerError("internal server error")
	}
}
