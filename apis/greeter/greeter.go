package greeter

import "context"

// Server implements the GreeterServer
type Server struct {
	UnimplementedGreeterServer
}

// SayHello - interface implementation
func (s *Server) SayHello(ctx context.Context, request *SayHelloRequest) (*SayHelloReply, error) {
	return &SayHelloReply{Message: "Hello, " + request.Name}, nil
}
