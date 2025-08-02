package str

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToScreamingSnakeCase(t *testing.T) {
	t.Run("it should convert camelCase to SCREAMING_SNAKE_CASE", func(t *testing.T) {
		// GIVEN
		input := "camelCase"

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "CAMEL_CASE", result)
	})

	t.Run("it should convert PascalCase to SCREAMING_SNAKE_CASE", func(t *testing.T) {
		// GIVEN
		input := "PascalCase"

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "PASCAL_CASE", result)
	})

	t.Run("it should handle lowercase with underscores", func(t *testing.T) {
		// GIVEN
		input := "lower_case_string"

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "LOWER_CASE_STRING", result)
	})

	t.Run("it should handle kebab-case", func(t *testing.T) {
		// GIVEN
		input := "kebab-case-string"

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "KEBAB_CASE_STRING", result)
	})

	t.Run("it should handle strings with numbers", func(t *testing.T) {
		// GIVEN
		input := "version2Release"

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "VERSION_2_RELEASE", result)
	})

	t.Run("it should handle consecutive uppercase letters", func(t *testing.T) {
		// GIVEN
		input := "XMLHttpRequest"

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "X_M_L_HTTP_REQUEST", result)
	})

	t.Run("it should handle single characters", func(t *testing.T) {
		// GIVEN
		input := "a"

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "A", result)
	})

	t.Run("it should handle empty string", func(t *testing.T) {
		// GIVEN
		input := ""

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "", result)
	})

	t.Run("it should handle strings with only spaces", func(t *testing.T) {
		// GIVEN
		input := "   "

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "", result)
	})

	t.Run("it should trim whitespace", func(t *testing.T) {
		// GIVEN
		input := "  camelCase  "

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "CAMEL_CASE", result)
	})

	t.Run("it should handle strings starting with numbers", func(t *testing.T) {
		// GIVEN
		input := "2ndVersion"

		// WHEN
		result := ToScreamingSnakeCase(input)

		// THEN
		assert.Equal(t, "2ND_VERSION", result)
	})

	t.Run("it should handle complex real-world examples", func(t *testing.T) {
		// GIVEN
		testCases := map[string]string{
			"customerId":     "CUSTOMER_ID",
			"XMLParser":      "X_M_L_PARSER",
			"httpStatusCode": "HTTP_STATUS_CODE",
			"fooBar":         "FOO_BAR",
			"FooBar":         "FOO_BAR",
			"foo_bar":        "FOO_BAR",
			"foo-bar":        "FOO_BAR",
			"API2Response":   "A_P_I_2_RESPONSE",
		}

		for input, expected := range testCases {
			// WHEN
			result := ToScreamingSnakeCase(input)

			// THEN
			assert.Equal(t, expected, result, "Failed for input: %s", input)
		}
	})
}
