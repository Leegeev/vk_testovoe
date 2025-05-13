package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"

	// "syscall"
	"time"

	pb "github.com/Leegeev/vk_testovoe/pkg/api"
	"github.com/Leegeev/vk_testovoe/pkg/config"
	"github.com/Leegeev/vk_testovoe/pkg/subpub"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	if err := config.InitConfig(); err != nil {
		log.Fatalf("Error occured while initializing configs %s", err.Error())
	}
	addr := viper.GetString("server.listen_addr")
	// 1) стартуем сервер
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(status.Errorf(
			codes.Unavailable,
			"cannot listen on port %v: %v",
			addr,
			err,
		))
	}

	// создаем контекст для отмены подписки
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// DI: Dependency Injection
	// 1. Создаём зависимости
	bus := subpub.NewSubPub()
	// 2. Внедряем в сервер
	srv := NewServer(bus, ctx)

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

	log.Println("Server started on: ", addr)

	// 2) ждём сигнал на shutdown
	<-ctx.Done()
	log.Println("Shutdown signal received")

	// 3) контекст с таймаутом
	timeoutDuration := viper.GetInt("server.shutdown_timeout_s")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutDuration)*time.Second)
	defer cancel()

	// 4) шина событий
	if err := srv.bus.Close(ctx); err != nil {
		log.Printf("SubPub shutdown incomplete: %v", err)
	} else {
		log.Println("SubPub shutdown complete")
	}
	// 5) gRPC graceful stop
	grpcServer.GracefulStop()
	log.Println("gRPC server stopped")

	log.Println("All done, exiting")
}
