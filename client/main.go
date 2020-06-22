package main

import (
	"context"
	"log"

	"github.com/alonlong/discovery/proto/greeter"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:15001", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := greeter.NewGreeterClient(conn)

	r, err := c.SayHello(context.Background(), &greeter.SayHelloRequest{Name: "Alon"})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
