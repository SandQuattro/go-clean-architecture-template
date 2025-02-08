package v1

import (
	"errors"

	"clean-arch-template/internal/usecase"

	"github.com/danielgtaylor/huma/v2"
)

// Errors for the handler package
var (
	ErrEmptyName = errors.New("name is required")
)

// MapError maps domain errors to HTTP errors
func MapError(err error) error {
	switch {
	case errors.Is(err, ErrEmptyName):
		return huma.Error400BadRequest(err.Error())
	case errors.Is(err, usecase.ErrUserNotFound):
		return huma.Error404NotFound(err.Error())
	default:
		return huma.Error500InternalServerError("internal server error: " + err.Error())
	}
}
