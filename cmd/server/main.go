package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/Leegeev/vk_testovoe/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	// 1) поднимаем TCP-лисенер на 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(status.Errorf(
			codes.Unavailable,
			"cannot listen on port 50051: %v",
			err,
		))
	}

	// 2) создаём gRPC-сервер
	grpcServer := grpc.NewServer()

	// 3) регистрируем наш сервис
	pb.RegisterPubSubServer(grpcServer, NewServer())

	log.Println("gRPC server listening on :50051")

	// 4) стартуем
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(status.Errorf(
			codes.Internal,
			"failed to serve gRPC server: %v",
			err,
		))
	}
}
