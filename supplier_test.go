package godi

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestToStaticProvider(t *testing.T) {
	t.Run("it should allow to register constant values", func(t *testing.T) {
		// GIVEN
		resolver := New()

		// WHEN
		resolver.MustRegister(
			ToStaticProvider("WE"),
			Named("env.prefix"),
		)
		resolver.MustRegister(
			ToStaticProvider(42),
			Named("answer.of.universe"),
		)

		// THEN
		strResolved, err := ResolveNamed[string](resolver, "env.prefix")
		require.NoError(t, err)
		require.Equal(t, "WE", strResolved)

		intResolved, err := ResolveNamed[int](resolver, "answer.of.universe")
		require.NoError(t, err)
		require.Equal(t, 42, intResolved)
	})
}
