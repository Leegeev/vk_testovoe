// Пакет main, потому что компилируем бинарник.
package main

import (
	"context"
	pb "github.com/Leegeev/vk_testovoe/pkg/api" // сгенерированный из pubsub.proto
	"github.com/Leegeev/vk_testovoe/pkg/subpub"
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
	// отправляем сообщение в шину
	if err := s.bus.Publish(req.Key, req.Data); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Subscribe — Server-stream RPC.
func (s *grpcPubSubServer) Subscribe(
	req *pb.SubscribeRequest,
	stream pb.PubSub_SubscribeServer,
) error {
	// заводим подписку — коллбэком шлём в stream
	sub, err := s.bus.Subscribe(req.Key, func(m interface{}) {
		// преобразуем интерфейс в string и отправляем
		_ = stream.Send(&pb.Event{Data: m.(string)})
	})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	// ждём, пока клиент не закроет stream.Context()
	<-stream.Context().Done()
	return stream.Context().Err()
}
