// Package rabbit contains tools for reading/writing and connecting to RabbitMQ
package rabbit

import (
	"context"

	"github.com/pquerna/ffjson/ffjson"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/moooll/cat-service-mongo/internal/config"
	"github.com/moooll/cat-service-mongo/internal/streams"
)

// NewChan returns newly created channel for RabbitMQ reading/writing
func NewChan(cfg config.Config) (*amqp.Channel, error) {
	conn, err := amqp.Dial(cfg.RabbitURI)
	if err != nil {
		return nil, err
	}

	ch, er := conn.Channel()
	if err != nil {
		return nil, er
	}

	return ch, nil
}

// WriteFromRedis reads from Redis Stream and writes to RabbitMQ
func WriteFromRedis(ss *streams.StreamService, ch *amqp.Channel) error {
	for {
		data, err := ss.Read(context.Background(), "$")
		if err != nil {
			return err
		}

		dataB, e := ffjson.Marshal(&data)
		if e != nil {
			return e
		}

		er := ch.Publish(
			"",
			"delete-cats",
			false,
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "plain/text",
				Body:         dataB,
			},
		)
		if er != nil {
			return er
		}
	}
}

// Read reads from RabbitMQ and prints the message
func Read(ch *amqp.Channel) error {
	deliv, err := ch.Consume(
		"delete-cats",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for v := range deliv {
			log.Infoln("Message from Rabbit: ", string(v.Body))
		}
	}()

	wait := make(chan bool)
	<-wait
	return err
}

// NewQueue returns new RabbitMQ queue and an error
func NewQueue(ch *amqp.Channel) (amqp.Queue, error) {
	q, err := ch.QueueDeclare(
		"delete-cats",
		true,
		false,
		false,
		false,
		nil,
	)
	return q, err
}
