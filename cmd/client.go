package cmd

import (
	"context"
	"discovery/apis"
	"log"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// cliCmd represents the serve command
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
	conn, err := grpc.Dial("localhost:15001", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := apis.NewGreeterServiceClient(conn)

	r, err := c.SayHello(context.Background(), &apis.SayHelloRequest{Name: "Alon"})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
