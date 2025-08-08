package main

import (
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_parseProperties(t *testing.T) {
	t.Run("it should parse simple key=value properties", func(t *testing.T) {
		// GIVEN
		line := "@provider named=foo priority=10"
		tag := "@provider"

		// WHEN
		result := parseProperties(line, tag)

		// THEN
		assert.Equal(t, "foo", result["named"])
		assert.Equal(t, "10", result["priority"])
	})

	t.Run("it should parse quoted values", func(t *testing.T) {
		// GIVEN
		line := `@provider named="hello world" priority=5`
		tag := "@provider"

		// WHEN
		result := parseProperties(line, tag)

		// THEN
		assert.Equal(t, "hello world", result["named"])
		assert.Equal(t, "5", result["priority"])
	})

	t.Run("it should return empty map for empty content", func(t *testing.T) {
		// GIVEN
		line := "@provider"
		tag := "@provider"

		// WHEN
		result := parseProperties(line, tag)

		// THEN
		assert.Empty(t, result)
	})
}

func Test_parseWhenAnnotation(t *testing.T) {
	t.Run("it should parse equals condition", func(t *testing.T) {
		// GIVEN
		logger := zerolog.Nop()
		line := `@when named="ENV" equals="production"`

		// WHEN
		result, err := parseWhenAnnotation(&logger, line)

		// THEN
		assert.NoError(t, err)
		assert.Equal(t, "ENV", result.named)
		assert.Equal(t, "equals", result.operator)
		assert.Equal(t, "production", result.value)
	})

	t.Run("it should parse not_equals condition", func(t *testing.T) {
		// GIVEN
		logger := zerolog.Nop()
		line := `@when named="DEBUG" not_equals="true"`

		// WHEN
		result, err := parseWhenAnnotation(&logger, line)

		// THEN
		assert.NoError(t, err)
		assert.Equal(t, "DEBUG", result.named)
		assert.Equal(t, "not_equals", result.operator)
		assert.Equal(t, "true", result.value)
	})

	t.Run("it should return error for missing named property", func(t *testing.T) {
		// GIVEN
		logger := zerolog.Nop()
		line := `@when equals="value"`

		// WHEN
		_, err := parseWhenAnnotation(&logger, line)

		// THEN
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'named' property")
	})

	t.Run("it should return error for missing operator", func(t *testing.T) {
		// GIVEN
		logger := zerolog.Nop()
		line := `@when named="ENV"`

		// WHEN
		_, err := parseWhenAnnotation(&logger, line)

		// THEN
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'equals' or 'not_equals'")
	})
}
