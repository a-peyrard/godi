package config

type FoobarConfig struct {
	Foo string
}

func (c *FoobarConfig) ApplyDefault() {
	if c.Foo == "" {
		c.Foo = "bar"
	}
}
