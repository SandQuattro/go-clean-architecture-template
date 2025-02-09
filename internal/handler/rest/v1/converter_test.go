package v1

import (
	"testing"

	"clean-arch-template/internal/entity"

	"github.com/stretchr/testify/assert"
)

func TestToUserListOutputFromEntity(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		users := []entity.User{}
		result := ToUserListOutputFromEntity(users)
		assert.Empty(t, result.Body.Users)
	})

	t.Run("list with users", func(t *testing.T) {
		users := []entity.User{
			{ID: 1, Name: "User 1"},
			{ID: 2, Name: "User 2"},
		}
		result := ToUserListOutputFromEntity(users)

		assert.Len(t, result.Body.Users, 2)
		assert.Equal(t, users[0].ID, result.Body.Users[0].ID)
		assert.Equal(t, users[0].Name, result.Body.Users[0].Name)
		assert.Equal(t, users[1].ID, result.Body.Users[1].ID)
		assert.Equal(t, users[1].Name, result.Body.Users[1].Name)
	})
}

func TestToUserOutputFromEntity(t *testing.T) {
	t.Run("valid user", func(t *testing.T) {
		user := &entity.User{
			ID:   1,
			Name: "Test User",
		}
		result := ToUserOutputFromEntity(user)

		assert.Equal(t, user.ID, result.Body.ID)
		assert.Equal(t, user.Name, result.Body.Name)
	})

	t.Run("nil user", func(t *testing.T) {
		result := ToUserOutputFromEntity(nil)
		assert.Nil(t, result.Body.User)
	})
}
