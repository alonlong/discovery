package cmd

import (
	"discovery/apis"
	"log"
	"net"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the grpc server",
	Run: func(cmd *cobra.Command, args []string) {
		// start the grpc server
		serve()
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
}

// the main process for the server subcommand
func serve() {
	lis, err := net.Listen("tcp", "localhost:15001")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	apis.RegisterGreeterServiceServer(s, &apis.GreeterService{})

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
