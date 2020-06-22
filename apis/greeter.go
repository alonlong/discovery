package apis

import "context"

// GreeterService implements the GreeterServiceServer
type GreeterService struct {
	UnimplementedGreeterServiceServer
}

// SayHello - interface implementation
func (s *GreeterService) SayHello(ctx context.Context, request *SayHelloRequest) (*SayHelloReply, error) {
	return &SayHelloReply{Message: "Hello, " + request.Name}, nil
}
