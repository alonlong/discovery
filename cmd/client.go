package cmd

import (
	"context"
	"discovery/apis/greeter"
	"discovery/pkg/etcd"
	"log"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
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
	cliCmd.PersistentFlags().StringVar(&addr, "addr", "localhost:2379", "etcd server's address")
}

func init() {
	RootCmd.AddCommand(cliCmd)
}

// the main process for the client subcommand
func cli() {
	etcd.NewRegister(addr)
	conn, err := grpc.Dial(
		"etcd://authority/my-service",
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
