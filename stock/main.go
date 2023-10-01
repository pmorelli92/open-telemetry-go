package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pmorelli92/bunnify/bunnify"
	"github.com/pmorelli92/open-telemetry-go/utils"
	"go.opentelemetry.io/otel"
)

func main() {
	jaegerEndpoint := utils.EnvString("JAEGER_ENDPOINT", "localhost:4318")
	amqpUser := utils.EnvString("RABBITMQ_USER", "guest")
	amqpPass := utils.EnvString("RABBITMQ_PASS", "guest")
	amqpHost := utils.EnvString("RABBITMQ_HOST", "localhost")
	amqpPort := utils.EnvString("RABBITMQ_PORT", "5672")
	amqpDNS := fmt.Sprintf("amqp://%s:%s@%s:%s", amqpUser, amqpPass, amqpHost, amqpPort)

	err := utils.SetGlobalTracer(context.Background(), "stock", jaegerEndpoint)
	if err != nil {
		log.Fatalf("failed to create tracer: %v", err)
	}

	cn := bunnify.NewConnection(bunnify.WithURI(amqpDNS))
	cn.Start()

	consumer := cn.NewConsumer(
		"stock-queue",
		bunnify.WithBindingToExchange("exchange"),
		bunnify.WithHandler[any]("checkout.processed", checkoutProcessedHandler))

	if err := consumer.Consume(); err != nil {
		log.Fatal(err)
	}

	log.Println("AMQP listening")

	// Block termination
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func checkoutProcessedHandler(ctx context.Context, event bunnify.ConsumableEvent[any]) error {
	// Create a new span
	tr := otel.Tracer("amqp")
	_, messageSpan := tr.Start(ctx, "AMQP - consume - checkout.processed")
	defer messageSpan.End()

	// Imaginary processing time
	time.Sleep(50 * time.Millisecond)

	return nil
}
