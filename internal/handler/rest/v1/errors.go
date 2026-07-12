package v1

import (
	"clean-arch-template/internal/entity"
	"errors"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
)

// MapError маппит доменные ошибки в HTTP-ошибки. Неизвестные ошибки логируются
// и уходят клиенту как generic 500 — текст внутренних ошибок не покидает сервис.
func MapError(err error) error {
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
		slog.Error("request failed", slog.String("error", err.Error()))
		return huma.Error500InternalServerError("internal server error")
	}
}
