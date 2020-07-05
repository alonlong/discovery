package cmd

import (
	"context"
	"discovery/apis/greeter"
	"discovery/pkg/balancer"
	"log"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

// cliCmd represents the client command
var cliCmd = &cobra.Command{
	Use:   "client",
	Short: "Request the grpc server",
	Run: func(cmd *cobra.Command, args []string) {
		cli()
	},
}

func init() {
	RootCmd.AddCommand(cliCmd)
}

// the main process for the client subcommand
func cli() {
	r := balancer.NewEtcdBalancer("localhost:2379").Resolver()
	resolver.Register(r)
	conn, err := grpc.Dial(
		r.Scheme()+"://author/my-service",
		grpc.WithBalancerName("round_robin"),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := greeter.NewGreeterClient(conn)

	for {
		reply, err := c.SayHello(context.Background(), &greeter.SayHelloRequest{Name: "Alon"})
		if err != nil {
			log.Printf("could not greet: %v", err)
		} else {
			log.Printf("Greeting: %s", reply.Message)
		}

		time.Sleep(time.Second)
	}
}
