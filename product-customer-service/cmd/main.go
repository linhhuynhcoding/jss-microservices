package main

import (
	"log"
	"net"

	"github.com/linhhuynhcoding/jss-microservices/rpc/gen/product"
	"google.golang.org/grpc"
)

func main() {
}

func NewServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	log.Println("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
