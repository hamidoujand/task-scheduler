package rabbitmq

import (
	"context"
	"fmt"
	"net/url"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQClient represents a set of APIs we need to access when working againt
// rabbitmq client.
type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// Configs represents all required configs for creating a rabbitmq client.
type Configs struct {
	Host     string
	User     string
	Password string
}

// NewClient creates a connection rabbitmq server and creates a client return it or possible error.
func NewClient(ctx context.Context, conf Configs) (*Client, error) {

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second*5)
		defer cancel()
	}

	url := url.URL{
		Scheme: "amqp",
		Host:   conf.Host,
		User:   url.UserPassword(conf.User, conf.Password),
	}

	var conn *amqp.Connection

	//we need a retry functionality
	for attemp := 1; ; attemp++ {
		var dialErr error
		conn, dialErr = amqp.Dial(url.String())
		if dialErr == nil {
			break
		}
		//sleep
		time.Sleep(time.Duration(attemp) * 100 * time.Millisecond)
		//check the ctx
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("dial: %w", dialErr)
		}
	}

	//check ctx again
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	//create a channel
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	return &Client{
		conn:    conn,
		channel: ch,
	}, nil
}

// Close will close the connection and the channel or returns possible errors
func (rc *Client) Close() error {
	err := rc.channel.Close()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}

	err = rc.conn.Close()
	if err != nil {
		return fmt.Errorf("connection: %w", err)
	}
	return nil
}

// DeclareQueue is going to create a queue to push messages into it.
func (rc *Client) DeclareQueue(name string) error {
	_, err := rc.channel.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declareQueue: %w", err)
	}
	return nil
}

// Publish enqueues the message into the queue or returns possible errors.
func (rc *Client) Publish(queue string, msg []byte) error {
	if err := rc.channel.Publish(
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         msg,
		},
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

// Consumer returns <-chan amqp.Delivery to consume messages from or possible error.
func (rc *Client) Consumer(queue string) (<-chan amqp.Delivery, error) {
	//limit the number of messages that the broker will deliver to consumers
	//before requiring an acknowledgment
	if err := rc.channel.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("qos: %w", err)
	}

	msgs, err := rc.channel.Consume(
		queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}

	return msgs, nil
}
