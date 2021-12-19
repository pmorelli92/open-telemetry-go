package utils

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
)

func ConnectAmqp(user, pass, host, port string) (*amqp.Channel, func() error) {
	address := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, pass, host, port)

	connection, err := amqp.Dial(address)
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
