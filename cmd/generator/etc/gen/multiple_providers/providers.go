package services

// @provider named="runner" priority=1
// DefaultRunner is the default runner implementation
func NewDefaultRunner() Runner {
	return &defaultRunner{}
}

// @provider named="runner" priority=100
// @when named="ENV" equals="dev"
// DevRunner is used in development
func NewDevRunner() Runner {
	return &devRunner{}
}

// @provider named="runner" priority=50
// @when named="ENV" equals="staging"
// StagingRunner is used in staging
func NewStagingRunner() Runner {
	return &stagingRunner{}
}

type Runner interface{}
type defaultRunner struct{}
type devRunner struct{}
type stagingRunner struct{}
