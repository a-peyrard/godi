package services

// DecorateWithLogging Adds logging to the hello service
//
// @decorator named="hello.service"
func DecorateWithLogging(
	service *HelloService,
	logger Logger, // @inject named="logger"
) *HelloService {
	// wrap with logging
	return service
}

type HelloService struct{}
type Logger interface{}
