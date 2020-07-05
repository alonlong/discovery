package cmd

import (
	"context"
	"discovery/pkg/balancer"
	"log"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/resolver"
)

// reflectCmd represents the serve command
var reflectCmd = &cobra.Command{
	Use:   "reflect",
	Short: "Reflect the grpc server",
	Run: func(cmd *cobra.Command, args []string) {
		reflect()
	},
}

func init() {
	RootCmd.AddCommand(reflectCmd)
}

// the main process for the reflect subcommand
func reflect() {
	r := balancer.NewEtcdBalancer("localhost:2379").Resolver()
	resolver.Register(r)
	conn, err := grpc.Dial(
		r.Scheme()+"://author/my-service",
		grpc.WithBalancerName("round_robin"),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()
	log.Printf("connected")

	reflectClient := grpcreflect.NewClient(context.Background(), reflectpb.NewServerReflectionClient(conn))
	defer reflectClient.Reset()

	sourceReflect := grpcurl.DescriptorSourceFromServer(context.Background(), reflectClient)
	services, err := grpcurl.ListServices(sourceReflect)
	if err != nil {
		log.Fatalf("list services: %v", err)
	}
	for _, service := range services {
		if service == "grpc.reflection.v1alpha.ServerReflection" {
			continue
		}
		log.Printf("service: %s", service)
		methods, err := grpcurl.ListMethods(sourceReflect, service)
		if err != nil {
			log.Fatalf("list methods: %v", err)
		}
		for _, method := range methods {
			log.Printf("\tmethod: %s", method)
		}
	}
}
