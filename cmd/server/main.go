package main

import (
	"log"
	"net"

	pb "github.com/Leegeev/vk_testovoe/pkg/api"
	"google.golang.org/grpc"
)

func main() {
	// 1) поднимаем TCP-лисенер на 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 2) создаём gRPC-сервер
	grpcServer := grpc.NewServer()

	// 3) регистрируем наш сервис
	pb.RegisterPubSubServer(grpcServer, NewServer())

	log.Println("gRPC server listening on :50051")
	// 4) стартуем
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
