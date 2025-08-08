package main

import (
	"github.com/a-peyrard/godi/set"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_findSuitableAlias(t *testing.T) {
	t.Run("it should find an alias", func(t *testing.T) {
		// GIVEN
		pkg := "github.com/a-peyrard/godi/fn"
		aliases := set.NewWithValues[string]()

		// WHEN
		alias := findSuitableAlias(pkg, aliases)

		// THEN
		assert.Equal(t, "fn", alias)
	})

	t.Run("it should use previous token if we have a collision", func(t *testing.T) {
		// GIVEN
		pkg := "github.com/a-peyrard/godi/fn"
		aliases := set.NewWithValues[string]("fn")

		// WHEN
		alias := findSuitableAlias(pkg, aliases)

		// THEN
		assert.Equal(t, "gfn", alias)
	})

	t.Run("it should use previous previous token if we have a collision", func(t *testing.T) {
		// GIVEN
		pkg := "github.com/a-peyrard/godi/fn"
		aliases := set.NewWithValues[string]("fn", "gfn")

		// WHEN
		alias := findSuitableAlias(pkg, aliases)

		// THEN
		assert.Equal(t, "agfn", alias)
	})

	t.Run("it should use exhaust all tokens if we have a collision", func(t *testing.T) {
		// GIVEN
		pkg := "github.com/a-peyrard/godi/fn"
		aliases := set.NewWithValues[string]("fn", "gfn", "agfn")

		// WHEN
		alias := findSuitableAlias(pkg, aliases)

		// THEN
		assert.Equal(t, "gagfn", alias)
	})

	t.Run("it should start incrementing when we don't token no more and still have a collision", func(t *testing.T) {
		// GIVEN
		pkg := "github.com/a-peyrard/godi/fn"
		aliases := set.NewWithValues[string]("fn", "gfn", "agfn", "gagfn", "gagfn0", "gagfn1")

		// WHEN
		alias := findSuitableAlias(pkg, aliases)

		// THEN
		assert.Equal(t, "gagfn2", alias)
	})
}

func Test_generateFQN(t *testing.T) {
	t.Run("it should return type name when import path is empty", func(t *testing.T) {
		// GIVEN
		importPath := ""
		typeName := "MyType"
		importWithAlias := map[string]string{}

		// WHEN
		result := generateFQN(importPath, typeName, importWithAlias)

		// THEN
		assert.Equal(t, "MyType", result)
	})

	t.Run("it should prepend alias for regular type", func(t *testing.T) {
		// GIVEN
		importPath := "github.com/example/pkg"
		typeName := "MyType"
		importWithAlias := map[string]string{
			"github.com/example/pkg": "pkg",
		}

		// WHEN
		result := generateFQN(importPath, typeName, importWithAlias)

		// THEN
		assert.Equal(t, "pkg.MyType", result)
	})

	t.Run("it should handle pointer types correctly", func(t *testing.T) {
		// GIVEN
		importPath := "github.com/example/pkg"
		typeName := "*MyType"
		importWithAlias := map[string]string{
			"github.com/example/pkg": "pkg",
		}

		// WHEN
		result := generateFQN(importPath, typeName, importWithAlias)

		// THEN
		assert.Equal(t, "*pkg.MyType", result)
	})
}
