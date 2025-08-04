package config

import (
	"fmt"
	"strings"

	"github.com/a-peyrard/godi/fn"
	"github.com/a-peyrard/godi/option"
	"github.com/a-peyrard/godi/reflectutils"
	"github.com/a-peyrard/godi/str"
	"github.com/spf13/viper"
	"reflect"
)

type (
	// Config represents a configuration instance backed by Viper
	Config struct {
		*viper.Viper
	}

	Options struct {
		prefix string
	}

	WithDefault interface {
		ApplyDefault()
	}
)

func WithEnvPrefix(prefix string) option.Option[Options] {
	return func(opts *Options) {
		opts.prefix = prefix
	}
}

func Load[T any](opts ...option.Option[Options]) (*T, error) {
	options := option.Build(&Options{}, opts...)

	v := viper.New()
	v.SetEnvPrefix(options.prefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var vT T
	bindEnvs(v, options.prefix, reflect.New(reflect.TypeOf(vT)).Elem().Interface())

	if err := v.Unmarshal(&vT); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config: %w", err)
	}

	withDefaultValueType := reflect.TypeOf((*WithDefault)(nil)).Elem()
	callApplyDefault := func(val reflect.Value, typ reflect.Type, _ []string) {
		if typ.Implements(withDefaultValueType) {
			if val.IsValid() {
				val.Interface().(WithDefault).ApplyDefault()
			}
		}
	}
	reflectutils.WalkStruct(
		&vT,
		fn.AllTriConsumer(
			reflectutils.CreateNilStructs,
			callApplyDefault,
		),
	)

	return &vT, nil
}

func bindEnvs(viperI *viper.Viper, envPrefix string, myStruct any, parts ...string) {
	ifv := reflect.ValueOf(myStruct)
	ift := reflect.TypeOf(myStruct)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			tv = t.Name
		}
		switch v.Kind() {
		case reflect.Struct:
			bindEnvs(viperI, envPrefix, v.Interface(), append(parts, tv)...)
		case reflect.Pointer:
			if t.Type.Elem().Kind() == reflect.Struct {
				bindEnvs(viperI, envPrefix, reflect.Zero(t.Type.Elem()).Interface(), append(parts, tv)...)
			}
		default:
			key := strings.Join(append(parts, tv), ".")
			join := strings.Join(append(parts, str.ToScreamingSnakeCase(tv)), ".")
			_ = viperI.BindEnv(key, mergeWithEnvPrefix(envPrefix, join))
		}
	}
}

func mergeWithEnvPrefix(envPrefix string, in string) string {
	if envPrefix != "" {
		return strings.ToUpper(envPrefix + "_" + in)
	}

	return strings.ToUpper(in)
}
