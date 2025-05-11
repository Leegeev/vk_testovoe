package main

import (
	"context"
	"fmt"
	"io"
	"log"

	pb "github.com/Leegeev/vk_testovoe/pkg/api"
	"google.golang.org/grpc"
)

// runPublish публикует одно сообщение и возвращается.
func runPublish(client pb.PubSubClient, key, msg string) {
	_, err := client.Publish(context.Background(), &pb.PublishRequest{
		Key:  key,
		Data: msg,
	})
	if err != nil {
		log.Fatalf("Publish error: %v", err)
	}
	fmt.Printf("-> published %q to %q\n", msg, key)
}

// runSubscribe подписывается и непрерывно читает из стрима.
func runSubscribe(client pb.PubSubClient, key string) {
	stream, err := client.Subscribe(context.Background(), &pb.SubscribeRequest{Key: key})
	if err != nil {
		log.Fatalf("Subscribe error: %v", err)
	}
	for {
		evt, err := stream.Recv()
		if err == io.EOF {
			log.Println("stream closed by server")
			return
		}
		if err != nil {
			log.Fatalf("Recv error: %v", err)
		}
		fmt.Printf("<- event on %q: %q\n", key, evt.Data)
	}
}
