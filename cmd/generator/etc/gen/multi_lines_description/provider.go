package registry

// NewHelloService provides a greeting service
//
// Is a service providing hello functionality.
// This service can be used to greet users.
// This is "really" a 'complex' service with multiple lines of description.
//
// @provider named="hello.service"
func NewHelloService() *HelloService {
	return &HelloService{}
}

type HelloService struct{}
