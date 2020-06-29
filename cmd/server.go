package cmd

import (
	"discovery/apis/greeter"
	"discovery/pkg/balancer"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	ip   string
	port string
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
	serveCmd.PersistentFlags().StringVar(&ip, "ip", "localhost", "grpc server's ip")
	serveCmd.PersistentFlags().StringVar(&port, "port", "15001", "grpc server's port")
}

func init() {
	RootCmd.AddCommand(serveCmd)
}

// init service information to register etcd
func newService() *balancer.Service {
	return &balancer.Service{
		ID:   uuid.New().String(),
		Name: "my-service",
		Endpoints: []balancer.Endpoint{
			{
				IP:       ip,
				Port:     port,
				Protocol: "GRPC",
				Version:  "v1.0.0",
				Metadata: map[string]string{"role": "service"},
			},
		},
	}
}

// the main process for the server subcommand
func serve() {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	greeter.RegisterGreeterServer(s, &greeter.Server{})

	// register the service to etcd registry
	etcdBalancer := balancer.NewEtcdBalancer("localhost:2379")

	var wg sync.WaitGroup
	wg.Add(1)
	go etcdBalancer.Register(&wg, newService())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		select {
		case <-sig:
			// unregister the service first
			if err := etcdBalancer.UnRegister(); err != nil {
				log.Printf("unregister: %v", err)
			}

			// close the etcd balancer
			etcdBalancer.Close()

			// stop the grpc server
			s.GracefulStop()

			return
		}
	}()

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	wg.Wait()
}
