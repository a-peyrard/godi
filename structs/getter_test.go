package structs

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGet(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}

	type User struct {
		Name    string
		Age     int
		Address *Address
		private string
	}

	t.Run("it should get simple field from struct", func(t *testing.T) {
		// GIVEN
		user := User{Name: "John", Age: 30}

		// WHEN
		value, err := Get(user, "Name")

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "John", value)
	})

	t.Run("it should get nested field from struct", func(t *testing.T) {
		// GIVEN
		user := User{
			Name: "John",
			Address: &Address{
				Street: "123 Main St",
				City:   "Springfield",
			},
		}

		// WHEN
		value, err := Get(user, "Address.Street")

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "123 Main St", value)
	})

	t.Run("it should get value from map", func(t *testing.T) {
		// GIVEN
		data := map[string]any{
			"user": map[string]any{
				"name": "Alice",
				"age":  25,
			},
		}

		// WHEN
		value, err := Get(data, "user.name")

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "Alice", value)
	})

	t.Run("it should handle mixed struct and map access", func(t *testing.T) {
		// GIVEN
		user := User{
			Name: "Bob",
			Address: &Address{
				Street: "456 Oak Ave",
			},
		}
		wrapper := map[string]any{
			"user": user,
		}

		// WHEN
		value, err := Get(wrapper, "user.Address.Street")

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "456 Oak Ave", value)
	})

	t.Run("it should return error for non-existent field", func(t *testing.T) {
		// GIVEN
		user := User{Name: "John"}

		// WHEN
		value, err := Get(user, "NonExistent")

		// THEN
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "field NonExistent not found in struct User")
	})

	t.Run("it should return error for non-existent map key", func(t *testing.T) {
		// GIVEN
		data := map[string]string{"foo": "bar"}

		// WHEN
		value, err := Get(data, "missing")

		// THEN
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "key missing not found in map")
	})

	t.Run("it should return error when trying to traverse nil pointer", func(t *testing.T) {
		// GIVEN
		user := User{Name: "John", Address: nil}

		// WHEN
		value, err := Get(user, "Address.Street")

		// THEN
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "encountered nil value")
	})

	t.Run("it should return error for private field", func(t *testing.T) {
		// GIVEN
		user := User{Name: "John", private: "secret"}

		// WHEN
		value, err := Get(user, "private")

		// THEN
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "is not exportable")
	})

	t.Run("it should return error for nil origin", func(t *testing.T) {
		// GIVEN
		var user *User

		// WHEN
		value, err := Get(user, "Name")

		// THEN
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "encountered nil value at token Name")
	})

	t.Run("it should return error for empty field path", func(t *testing.T) {
		// GIVEN
		user := User{Name: "John"}

		// WHEN
		value, err := Get(user, "")

		// THEN
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "field path cannot be empty")
	})

	t.Run("it should return error for empty token in path", func(t *testing.T) {
		// GIVEN
		user := User{Name: "John"}

		// WHEN
		value, err := Get(user, "Name..Age")

		// THEN
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "empty token")
	})

	t.Run("it should return error when trying to traverse primitive type", func(t *testing.T) {
		// GIVEN
		user := User{Name: "John"}

		// WHEN
		value, err := Get(user, "Name.Length")

		// THEN
		assert.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "expected struct or map but got string")
	})
}
