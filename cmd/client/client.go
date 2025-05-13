package main

import (
	"context"
	"fmt"
	"io"
	"log"

	pb "github.com/Leegeev/vk_testovoe/pkg/api"
	_ "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// runPublish публикует одно сообщение и возвращается.
func runPublish(client pb.PubSubClient, key, msg string) {
	_, err := client.Publish(context.Background(), &pb.PublishRequest{
		Key:  key,
		Data: msg,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			log.Fatalf("non-gRPC error: %v", err)
		}

		switch st.Code() {
		case codes.NotFound:
			// ErrSubjectNotFound
			log.Fatalf("topic not found: %s", st.Message())
		case codes.Unavailable:
			// ErrClosed или проблемы с сетью
			log.Fatalf("service unavailable: %s", st.Message())
		default:
			log.Fatalf("publish error [%s]: %s", st.Code(), st.Message())
		}
		return
	}

	fmt.Printf("-> published %q to %q\n", msg, key)
}

// runSubscribe подписывается и непрерывно читает из стрима.
func runSubscribe(ctx context.Context, client pb.PubSubClient, key string) {
	stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{Key: key})
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			log.Fatalf("non-gRPC error on Subscribe: %v", err)
		}
		switch st.Code() {
		case codes.InvalidArgument:
			log.Fatalf("invalid argument: %s", st.Message())
		case codes.Unavailable:
			log.Fatalf("service unavailable: %s", st.Message())
		default:
			log.Fatalf("Subscribe error [%s]: %s", st.Code(), st.Message())
		}
	}
	defer func() {
		// при выходе из функции мы уже отменили контекст в main
		log.Println("Unsubscribing and closing stream")
	}()
	for {
		evt, err := stream.Recv()
		if err == io.EOF {
			log.Println("stream closed by server")
			return
		}
		if err != nil {
			if ctx.Err() != nil {
				log.Println("subscription cancelled")
				return
			}
			log.Fatalf("Recv error: %v", err)
		}
		fmt.Printf("<- event on %q: %q\n", key, evt.Data)
	}
}
