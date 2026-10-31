package v1

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

// Операции без тела запроса не могут вернуть 400: параметры пути валидирует
// схема (422), а 400 Huma отдаёт только на неразбираемое тело. Объявленный
// 400 в спецификации вводил бы клиентов в заблуждение.
func TestOpenAPIBodylessOperationsDoNotDeclareBadRequest(t *testing.T) {
	_, api := humatest.New(t, SetupHumaConfig())
	SetupRoutes(api, NewUserHandler(nil, nil))

	paths := api.OpenAPI().Paths
	ops := map[string]*huma.Operation{
		"GET /users/{page}/{size}":             paths["/users/{page}/{size}"].Get,
		"GET /users-with-orders/{page}/{size}": paths["/users-with-orders/{page}/{size}"].Get,
		"GET /user/{id}":                       paths["/user/{id}"].Get,
		"DELETE /user/{id}":                    paths["/user/{id}"].Delete,
	}

	for name, op := range ops {
		if _, ok := op.Responses["400"]; ok {
			t.Errorf("%s declares 400, but it is unreachable without a request body", name)
		}
	}
}
