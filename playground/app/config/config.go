package config

// Config contains the playground configuration.
//
// @config prefix=PG
type Config struct {
	Environment        string `mapstructure:"env" validate:"required"`
	EnvPrefixSeparator string
	Foobar             *FoobarConfig
}

func (c *Config) ApplyDefault() {}
