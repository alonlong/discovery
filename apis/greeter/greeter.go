package greeter

import (
	"context"

	uuid "github.com/satori/go.uuid"
)

// Server implements the GreeterServer
type Server struct {
	UnimplementedGreeterServer
}

// SayHello - interface implementation
func (s *Server) SayHello(ctx context.Context, request *SayHelloRequest) (*SayHelloReply, error) {
	return &SayHelloReply{Message: "Hello, " + request.Name}, nil
}

// Join - interface implementation
func (s *Server) Join(ctx context.Context, request *JoinRequest) (*JoinReply, error) {
	return &JoinReply{Id: uuid.NewV4().String()}, nil
}
