package godi

import "github.com/a-peyrard/godi/option"

type (
	condition struct {
		namedStringComponent string
		operator             operator
		value                string
	}

	operator = func(string, string) bool

	ConditionBuilder     struct{}
	ConditionNameBuilder struct {
		namedStringComponent string
	}
)

//goland:noinspection GoVarAndConstTypeMayBeOmitted
var (
	equals operator = func(a, b string) bool {
		return a == b
	}

	notEquals operator = func(a, b string) bool {
		return a != b
	}
)

func When(namedStringComponent string) ConditionNameBuilder {
	return ConditionNameBuilder{
		namedStringComponent: namedStringComponent,
	}
}

func (cn ConditionNameBuilder) Equals(value string) option.Option[RegistrableOptions] {
	return func(opts *RegistrableOptions) {
		opts.conditions = append(
			opts.conditions,
			condition{
				namedStringComponent: cn.namedStringComponent,
				operator:             equals,
				value:                value,
			},
		)
	}
}

func (cn ConditionNameBuilder) NotEquals(value string) option.Option[RegistrableOptions] {
	return func(opts *RegistrableOptions) {
		opts.conditions = append(
			opts.conditions,
			condition{
				namedStringComponent: cn.namedStringComponent,
				operator:             notEquals,
				value:                value,
			},
		)
	}
}
