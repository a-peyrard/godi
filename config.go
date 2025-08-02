package godi

import (
	"fmt"
	"github.com/a-peyrard/godi/structs"
)

type ConfigProvider[C any, T any] = func(cfg C) (T, error)

func ProvidesConfig[C any, T any](configPath string) ConfigProvider[C, T] {
	return func(cfg C) (v T, err error) {
		raw, err := structs.Get(cfg, configPath)
		if err != nil {
			return v, fmt.Errorf("Unable to get value from config %T:\n\t%w", cfg, err)
		}
		value, ok := raw.(T)
		if !ok {
			return v, fmt.Errorf("config value at %s is not of type %T", configPath, v)
		}
		return value, nil
	}
}
