package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	TestConfig struct {
		Foo *FooTestConfig
		Bar *BarTestConfig
	}
	FooTestConfig struct {
		Hello string
		World int
	}
	BarTestConfig struct {
		First  int
		Second int
	}
	MultipleWordsConfig struct {
		FooBar     int
		CustomerId int
	}
)

func (c *BarTestConfig) ApplyDefault() {
	if c.First == 0 {
		c.First = 42
	}
}

func TestLoad(t *testing.T) {
	t.Run("it should load basic struct", func(t *testing.T) {
		// GIVEN
		t.Setenv("FOO_HELLO", "waldo")
		t.Setenv("FOO_WORLD", "23")

		// WHEN
		conf, err := Load[FooTestConfig](WithEnvPrefix("FOO"))

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "waldo", conf.Hello)
		assert.Equal(t, 23, conf.World)
	})

	t.Run("it should load from env vars", func(t *testing.T) {
		// GIVEN
		t.Setenv("TEST_FOO_HELLO", "waldo")
		t.Setenv("TEST_FOO_WORLD", "23")
		t.Setenv("TEST_BAR_FIRST", "12")
		t.Setenv("TEST_BAR_SECOND", "66")

		// WHEN
		conf, err := Load[TestConfig](WithEnvPrefix("TEST"))

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "waldo", conf.Foo.Hello)
		assert.Equal(t, 23, conf.Foo.World)
		assert.Equal(t, 12, conf.Bar.First)
		assert.Equal(t, 66, conf.Bar.Second)
	})

	t.Run("it should initialize nested struct event if no env vars for this struct", func(t *testing.T) {
		// GIVEN
		t.Setenv("TEST_BAR_FIRST", "12")
		t.Setenv("TEST_BAR_SECOND", "66")

		// WHEN
		conf, err := Load[TestConfig](WithEnvPrefix("TEST"))

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "", conf.Foo.Hello)
		assert.Equal(t, 0, conf.Foo.World)
		assert.Equal(t, 12, conf.Bar.First)
		assert.Equal(t, 66, conf.Bar.Second)
	})

	t.Run("it should apply default if the struct implements WithDefault", func(t *testing.T) {
		// GIVEN

		// WHEN
		conf, err := Load[TestConfig](WithEnvPrefix("TEST"))

		// THEN
		require.NoError(t, err)
		assert.Equal(t, "", conf.Foo.Hello)
		assert.Equal(t, 0, conf.Foo.World)
		assert.Equal(t, 42, conf.Bar.First)
		assert.Equal(t, 0, conf.Bar.Second)
	})

	t.Run("it should bind correctly multiple words variables", func(t *testing.T) {
		// GIVEN
		t.Setenv("TEST_FOO_BAR", "12")
		t.Setenv("TEST_CUSTOMER_ID", "66")

		// WHEN
		conf, err := Load[MultipleWordsConfig](WithEnvPrefix("TEST"))

		// THEN
		require.NoError(t, err)
		assert.Equal(t, 12, conf.FooBar)
		assert.Equal(t, 66, conf.CustomerId)
	})
}
