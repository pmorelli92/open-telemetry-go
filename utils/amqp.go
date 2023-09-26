package utils

import (
	"fmt"
	"log"

	"github.com/rabbitmq/amqp091-go"
)

func ConnectAmqp(user, pass, host, port string) (*amqp091.Channel, func() error) {
	address := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, pass, host, port)

	connection, err := amqp091.Dial(address)
	if err != nil {
		log.Fatal(err)
	}

	channel, err := connection.Channel()
	if err != nil {
		log.Fatal(err)
	}

	err = channel.ExchangeDeclare("exchange", "direct", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	return channel, connection.Close
}
