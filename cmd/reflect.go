package cmd

import (
	"bytes"
	"context"
	"discovery/pkg/balancer"
	"encoding/json"
	"log"
	"strings"

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

	// list the services
	services, err := grpcurl.ListServices(sourceReflect)
	if err != nil {
		log.Fatalf("list services: %v", err)
	}
	for _, service := range services {
		if service == "grpc.reflection.v1alpha.ServerReflection" {
			continue
		}
		log.Printf("service: %s", service)

		// list the methods for service
		methods, err := grpcurl.ListMethods(sourceReflect, service)
		if err != nil {
			log.Fatalf("list methods: %v", err)
		}
		for _, method := range methods {
			log.Printf("\tmethod: %s", method)
		}

		descriptor, err := sourceReflect.FindSymbol(service)
		if err != nil {
			log.Fatalf("find symbol: %v", err)
		}
		log.Printf("fully qualified name: %s", descriptor.GetFullyQualifiedName())
	}

	// new reader for json string
	in := strings.NewReader(`{"name": "World"}`)
	parser, formmater, err := grpcurl.RequestParserAndFormatterFor(grpcurl.FormatJSON, sourceReflect, true, false, in)
	if err != nil {
		log.Fatalf("request parser and formmater: %v", err)
	}

	var out bytes.Buffer
	handler := grpcurl.NewDefaultEventHandler(&out, sourceReflect, formmater, false)

	symbol := "apis.Greeter/SayHello"
	if err := grpcurl.InvokeRPC(context.Background(), sourceReflect, conn, symbol, nil, handler, parser.Next); err != nil {
		log.Fatalf("invoke rpc: %v", err)
	}
	log.Printf("response: %s", out.String())
}

func struct2JSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
