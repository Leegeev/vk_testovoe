package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/Leegeev/vk_testovoe/pkg/api"
	"github.com/Leegeev/vk_testovoe/pkg/config"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var (
	mode = flag.String("mode", "sub", "pub или sub")
	key  = flag.String("key", "default", "subject key")
	msg  = flag.String("msg", "", "сообщение для pub")
)

func main() {
	flag.Parse()
	if err := config.InitConfig(); err != nil {
		log.Fatalf("Error occured while initializing configs %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	addr := viper.GetString("server.listen_addr")
	// 1) подключаемся к серверу
	conn, err := grpc.NewClient("localhost"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(status.Errorf(
			codes.Unavailable,
			"cannot connect to gRPC server: %v",
			err,
		))
	}
	defer conn.Close()
	client := pb.NewPubSubClient(conn)

	// 2) решаем, что делаем
	go func() {
		switch *mode {
		case "pub":
			runPublish(client, *key, *msg)
			cancel() // после публикации можно завершить
		case "sub":
			runSubscribe(ctx, client, *key)
			cancel() // после отписки — тоже выйти из main
		default:
			log.Fatalf("неизвестный режим %q: используйте pub или sub", *mode)
		}
	}()

	select {
	case <-stop:
		log.Println("Signal received, shutting down…")
		cancel()
	case <-ctx.Done():
	}
	log.Println("Client exiting")
}
