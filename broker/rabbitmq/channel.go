package rabbitmq

import (
	"errors"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type rabbitChannel struct {
	uuid       string
	connection *amqp.Connection
	channel    *amqp.Channel
}

func newRabbitChannel(conn *amqp.Connection, prefetchCount int, prefetchGlobal bool) (*rabbitChannel, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	rabbitCh := &rabbitChannel{
		uuid:       id.String(),
		connection: conn,
	}
	if err := rabbitCh.Connect(prefetchCount, prefetchGlobal); err != nil {
		return nil, err
	}
	return rabbitCh, nil
}

func (r *rabbitChannel) Connect(prefetchCount int, prefetchGlobal bool) error {
	var err error
	r.channel, err = r.connection.Channel()
	if err != nil {
		return err
	}
	err = r.channel.Qos(prefetchCount, 0, prefetchGlobal)
	if err != nil {
		return err
	}
	return nil
}

func (r *rabbitChannel) Close() error {
	if r.channel == nil {
		return errors.New("channel is nil")
	}
	return r.channel.Close()
}

func (r *rabbitChannel) Publish(exchange, key string, message amqp.Publishing) error {
	if r.channel == nil {
		return errors.New("channel is nil")
	}
	return r.channel.Publish(exchange, key, false, false, message)
}

func (r *rabbitChannel) DeclareExchange(exchange string) error {
	return r.channel.ExchangeDeclare(
		exchange, // name
		"topic",  // kind
		false,    // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	)
}

func (r *rabbitChannel) DeclareDurableExchange(exchange string) error {
	return r.channel.ExchangeDeclare(
		exchange, // name
		"topic",  // kind
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	)
}

func (r *rabbitChannel) DeclareQueue(queue string, args amqp.Table) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		false, // durable
		true,  // autoDelete
		false, // exclusive
		false, // noWait
		args,  // args
	)
	return err
}

func (r *rabbitChannel) DeclareDurableQueue(queue string, args amqp.Table) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		args,  // args
	)
	return err
}

func (r *rabbitChannel) DeclareReplyQueue(queue string) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		false, // durable
		true,  // autoDelete
		true,  // exclusive
		false, // noWait
		nil,   // args
	)
	return err
}

func (r *rabbitChannel) ConsumeQueue(queue string, autoAck bool) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(
		queue,   // queue
		r.uuid,  // consumer
		autoAck, // autoAck
		false,   // exclusive
		false,   // nolocal
		false,   // nowait
		nil,     // args
	)
}

func (r *rabbitChannel) BindQueue(queue, key, exchange string, args amqp.Table) error {
	return r.channel.QueueBind(
		queue,    // name
		key,      // key
		exchange, // exchange
		false,    // noWait
		args,     // args
	)
}
