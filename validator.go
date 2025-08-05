package godi

import (
	"fmt"
)

type (
	validator interface {
		validate(results []*queryResult) error

		fmt.Stringer
	}

	validatorUniqueMandatory struct{}

	validatorUniqueOptional struct{}

	validatorMultiple struct{}
)

func (c validatorUniqueMandatory) validate(results []*queryResult) error {
	if len(results) == 0 {
		return fmt.Errorf("no providers found for %s", c)
	}
	if len(results) > 1 {
		return fmt.Errorf("multiple providers found for %s, expected one and only one, got %d", c, len(results))
	}

	return nil
}

func (c validatorUniqueMandatory) String() string {
	return "<unique mandatory>"
}

func (c validatorUniqueOptional) validate(results []*queryResult) error {
	if len(results) > 1 {
		return fmt.Errorf("multiple providers found for %s, expected one and only one, got %d", c, len(results))
	}

	return nil
}

func (c validatorUniqueOptional) String() string {
	return "<unique optional>"
}

func (c validatorMultiple) validate([]*queryResult) error {
	return nil
}

func (c validatorMultiple) String() string {
	return "<multiple>"
}
