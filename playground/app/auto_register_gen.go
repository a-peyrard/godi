package main

import (
	"github.com/a-peyrard/godi"
	"github.com/a-peyrard/godi/playground/app/hello"
)

func AutoRegisterComponents(resolver *godi.Resolver) {
	resolver.MustRegister(hello.NewHelloRunner)
}
