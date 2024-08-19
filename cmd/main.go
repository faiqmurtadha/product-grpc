package main

import (
	"log"
	"net"

	"product-grpc/config"
	pb "product-grpc/protos/compiled"
	"product-grpc/services"

	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

func main() {

	netListen, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen %v", err.Error())
	}

	db := config.InitMongoDB()

	grpcServer := grpc.NewServer()
	productService := services.ProductService{DB: db}
	pb.RegisterProductServiceServer(grpcServer, &productService)

	log.Printf("Server started at %v", netListen.Addr())
	if err := grpcServer.Serve(netListen); err != nil {
		log.Fatalf("Failed to serve %v", err.Error())
	}
}
