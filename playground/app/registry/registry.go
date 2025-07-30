package registry

import "github.com/a-peyrard/godi"

//go:generate go run github.com/a-peyrard/godi/cmd/generator
type Registry struct {
	godi.EmptyRegistry
}
