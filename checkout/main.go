package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/pmorelli92/bunnify/bunnify"
	pb "github.com/pmorelli92/open-telemetry-go/proto"
	"github.com/pmorelli92/open-telemetry-go/utils"
	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func main() {
	jaegerEndpoint := utils.EnvString("JAEGER_ENDPOINT", "localhost:4318")
	grpcAddress := utils.EnvString("GRPC_ADDRESS", "localhost:8080")
	amqpUser := utils.EnvString("RABBITMQ_USER", "guest")
	amqpPass := utils.EnvString("RABBITMQ_PASS", "guest")
	amqpHost := utils.EnvString("RABBITMQ_HOST", "localhost")
	amqpPort := utils.EnvString("RABBITMQ_PORT", "5672")
	amqpDNS := fmt.Sprintf("amqp://%s:%s@%s:%s", amqpUser, amqpPass, amqpHost, amqpPort)

	err := utils.SetGlobalTracer(context.Background(), "checkout", jaegerEndpoint)
	if err != nil {
		log.Fatalf("failed to create tracer: %v", err)
	}

	cn := bunnify.NewConnection(bunnify.WithURI(amqpDNS))
	cn.Start()

	publisher := cn.NewPublisher()

	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()))

	pb.RegisterCheckoutServer(s, &server{publisher: publisher})

	log.Printf("GRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type server struct {
	pb.UnimplementedCheckoutServer
	publisher *bunnify.Publisher
}

func (s *server) DoCheckout(ctx context.Context, rq *pb.CheckoutRequest) (*pb.CheckoutResponse, error) {
	messageName := "checkout.processed"

	// Create a new span (child of the trace id) to inform the publishing of the message
	tr := otel.Tracer("amqp")
	amqpContext, messageSpan := tr.Start(ctx, fmt.Sprintf("AMQP - publish - %s", messageName))
	defer messageSpan.End()

	err := s.publisher.Publish(
		amqpContext, "exchange", messageName,
		bunnify.NewPublishableEvent(struct{}{}),
		func(p *amqp091.Publishing) {
			// Inject the context in the headers
			p.Headers = utils.InjectAMQPHeaders(amqpContext)
		})
	if err != nil {
		return nil, err
	}

	response := &pb.CheckoutResponse{TotalAmount: 1234}

	// Example on how to log specific events for a span
	span := trace.SpanFromContext(ctx)
	span.AddEvent(fmt.Sprintf("response: %v", response))

	return response, nil
}
