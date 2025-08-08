package registry

// @provider named="hello.service"
// HelloService provides a greeting service
func NewHelloService() *HelloService {
	return &HelloService{}
}

type HelloService struct{}
