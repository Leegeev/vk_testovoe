// Пакет main, потому что компилируем бинарник.
package main

import (
	"context"
	"errors"
	"log"

	pb "github.com/Leegeev/vk_testovoe/pkg/api" // сгенерированный из pubsub.proto
	"github.com/Leegeev/vk_testovoe/pkg/subpub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// grpcPubSubServer реализует auto-generated интерфейс pb.PubSubServer.
type grpcPubSubServer struct {
	pb.UnimplementedPubSubServer               // встраиваем заглушки
	bus                          subpub.SubPub // наша реализация шины
}

// NewServer создаёт и настраивает gRPC-сервис.
func NewServer() *grpcPubSubServer {
	return &grpcPubSubServer{
		bus: subpub.NewSubPub(),
	}
}

// Publish — классический Unary RPC.
func (s *grpcPubSubServer) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	if err := s.bus.Publish(req.Key, req.Data); err != nil {
		// 2) тема не найдена → NotFound
		if errors.Is(err, subpub.ErrTopictNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		// 3) шина уже закрыта → Unavailable
		if errors.Is(err, subpub.ErrClosed) {
			return nil, status.Error(codes.Unavailable, err.Error())
		}
		// 4) всё остальное → Internal
		return nil, status.Errorf(codes.Internal, "publish failed: %v", err)
	}
	log.Printf("Published \"%v\" in topic: %v", req.Data, req.Key)
	return &emptypb.Empty{}, nil
}

// Subscribe — Server-stream RPC.
func (s *grpcPubSubServer) Subscribe(req *pb.SubscribeRequest, stream pb.PubSub_SubscribeServer) error {
	// заводим подписку — коллбэком шлём в stream
	sub, err := s.bus.Subscribe(req.Key, func(m interface{}) {
		// преобразуем интерфейс в string и отправляем
		_ = stream.Send(&pb.Event{Data: m.(string)})
	})

	if err != nil {
		if errors.Is(err, subpub.ErrClosed) {
			return status.Error(codes.Unavailable, err.Error())
		}
		if errors.Is(err, subpub.ErrTopicNameIsEmpty) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "subscription failed: %v", err)
	}
	defer func() {
		sub.Unsubscribe()
		log.Printf("Client unsubscribed from topic %q", req.Key)
	}()

	// ждём, пока клиент не закроет stream.Context()
	log.Printf("New sub on topic: %v", req.Key)
	<-stream.Context().Done()
	return stream.Context().Err()
}
