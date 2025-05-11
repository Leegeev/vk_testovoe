package main

import (
	"flag"
	"log"

	pb "github.com/Leegeev/vk_testovoe/pkg/api"
	"google.golang.org/grpc"
)

var (
	mode = flag.String("mode", "sub", "pub или sub")
	key  = flag.String("key", "default", "subject key")
	msg  = flag.String("msg", "", "сообщение для pub")
)

func main() {
	flag.Parse()

	// 1) подключаемся к серверу
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()
	client := pb.NewPubSubClient(conn)

	// 2) решаем, что делаем
	switch *mode {
	case "pub":
		if *msg == "" {
			log.Fatalf("для pub нужно задать -msg")
		}
		runPublish(client, *key, *msg)

	case "sub":
		runSubscribe(client, *key)

	default:
		log.Fatalf("неизвестный режим %q", *mode)
	}
}
