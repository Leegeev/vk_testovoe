package main

import (
	"flag"
	"log"

	pb "github.com/Leegeev/vk_testovoe/pkg/api"
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

	// 1) подключаемся к серверу
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	switch *mode {
	case "pub":
		runPublish(client, *key, *msg)

	case "sub":
		runSubscribe(client, *key)

	default:
		err := status.Errorf(codes.InvalidArgument, "неизвестный режим %q: используйте pub или sub", *mode)
		log.Fatal(err)
	}
}
