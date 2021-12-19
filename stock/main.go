package main

import (
	"context"
	"github.com/pmorelli92/open-telemetry-go/utils"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"log"
	"sync"
	"time"
)

func main() {
	jaegerAddress := utils.EnvString("JAEGER_ADDRESS", "localhost")
	jaegerPort := utils.EnvString("JAEGER_PORT", "6831")
	amqpUser := utils.EnvString("RABBITMQ_USER", "guest")
	amqpPass := utils.EnvString("RABBITMQ_PASS", "guest")
	amqpHost := utils.EnvString("RABBITMQ_HOST", "localhost")
	amqpPort := utils.EnvString("RABBITMQ_PORT", "5672")

	err := utils.SetGlobalTracer("stock", jaegerAddress, jaegerPort)
	if err != nil {
		log.Fatalf("failed to create tracer: %v", err)
	}

	channel, closeConn := utils.ConnectAmqp(amqpUser, amqpPass, amqpHost, amqpPort)
	defer closeConn()

	// Create queue and binding
	_, err = channel.QueueDeclare("stock-queue", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = channel.QueueBind("stock-queue", "checkout.processed", "exchange", false, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Start consuming
	go ConsumeFromAMQP(channel)
	log.Println("AMQP listening")

	// Block termination
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func ConsumeFromAMQP(channel *amqp.Channel) {
	// Start the consumption
	deliveries, err := channel.Consume("stock-queue", "some-tag", false, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		// For each message
		case d := <-deliveries:

			// Extract headers
			ctx := utils.ExtractAMQPHeaders(context.Background(), d.Headers)

			// Create a new span
			tr := otel.Tracer("amqp")
			ctx, messageSpan := tr.Start(ctx, "AMQP - consume - checkout.processed")

			// Cannot use defer inside a for loop
			time.Sleep(1 * time.Millisecond)
			messageSpan.End()

			// ACK the message
			err = d.Ack(false)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
