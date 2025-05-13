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
	"github.com/Leegeev/vk_testovoe/pkg/subpub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	// 1) стартуем сервер
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(status.Errorf(
			codes.Unavailable,
			"cannot listen on port 50051: %v",
			err,
		))
	}

	// DI: Dependency Injection
	// 1. Создаём зависимости
	bus := subpub.NewSubPub()
	// 2. Внедряем в сервер
	srv := NewServer(bus)

	grpcServer := grpc.NewServer()
	pb.RegisterPubSubServer(grpcServer, srv)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(status.Errorf(
				codes.Internal,
				"failed to serve gRPC server: %v",
				err,
			))
		}
	}()

	log.Println("Server started on: 50051")

	// 2) ждём сигнал на shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutdown signal received")

	// 3) контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 4) сначала gRPC graceful stop
	grpcServer.GracefulStop()
	log.Println("gRPC server stopped")

	// 5) потом шина событий
	if err := srv.bus.Close(ctx); err != nil {
		log.Printf("SubPub shutdown incomplete: %v", err)
	} else {
		log.Println("SubPub shutdown complete")
	}

	log.Println("All done, exiting")
}
