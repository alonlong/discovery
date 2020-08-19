package cmd

import (
	"bytes"
	"context"
	"discovery/pkg/etcd"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// proxyCmd represents the serve command
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Proxy the http -> grpc server",
	Run: func(cmd *cobra.Command, args []string) {
		proxy()
	},
}

func init() {
	proxyCmd.PersistentFlags().StringVar(&addr, "addr", "localhost:2379", "etcd server's address")
}

func init() {
	RootCmd.AddCommand(proxyCmd)
}

var sourceReflect grpcurl.DescriptorSource
var conn *grpc.ClientConn

// start the proxy server
func proxy() {
	r := etcd.NewRegister(addr)
	c, err := grpc.Dial(
		r.Scheme()+"://author/my-service",
		grpc.WithBalancerName("round_robin"),
		grpc.WithInsecure(),
	)
	conn = c
	if err != nil {
		log.Fatalf("grpc dial: %v", err)
	}
	defer c.Close()

	reflectClient := grpcreflect.NewClient(context.Background(), reflectpb.NewServerReflectionClient(conn))
	defer reflectClient.Reset()

	sourceReflect = grpcurl.DescriptorSourceFromServer(context.Background(), reflectClient)

	http.HandleFunc("/api/", SayHello)
	http.ListenAndServe(":3000", nil)
}

// SayHello for path '/apis.Greeter/SayHello'
func SayHello(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimPrefix(r.URL.Path, "/api/")
	// read the http body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("ioutil read: %v", err)
	}
	defer r.Body.Close()

	// new reader for json body
	in := bytes.NewReader(data)
	parser, formmater, err := grpcurl.RequestParserAndFormatterFor(grpcurl.FormatJSON, sourceReflect, true, false, in)
	if err != nil {
		log.Fatalf("request parser and formmater: %v", err)
	}

	var out bytes.Buffer
	// invoke the rpc request to server
	handler := grpcurl.NewDefaultEventHandler(&out, sourceReflect, formmater, false)
	if err := grpcurl.InvokeRPC(context.Background(), sourceReflect, conn, symbol, nil, handler, parser.Next); err != nil {
		log.Fatalf("invoke rpc: %v", err)
	}

	fmt.Fprint(w, out.String())
}
