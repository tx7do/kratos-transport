package rabbitmq

import (
	"context"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitChannel struct {
	uuid       string
	connection *amqp.Connection
	channel    *amqp.Channel
}

func newRabbitChannel(conn *amqp.Connection, qos Qos) (*rabbitChannel, error) {
	rabbitCh := &rabbitChannel{
		uuid:       generateUUID(),
		connection: conn,
	}
	if err := rabbitCh.Connect(qos.PrefetchCount, qos.PrefetchSize, qos.PrefetchGlobal); err != nil {
		return nil, err
	}
	return rabbitCh, nil
}

func (r *rabbitChannel) Connect(prefetchCount, prefetchSize int, prefetchGlobal bool) error {
	var err error
	r.channel, err = r.connection.Channel()
	if err != nil {
		return err
	}
	err = r.channel.Qos(prefetchCount, prefetchSize, prefetchGlobal)
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

func (r *rabbitChannel) Publish(ctx context.Context, exchangeName, key string, message amqp.Publishing) error {
	if r.channel == nil {
		return errors.New("channel is nil")
	}
	return r.channel.PublishWithContext(ctx, exchangeName, key, false, false, message)
}

func (r *rabbitChannel) DeclareExchange(exchangeName, kind string, durable, autoDelete bool) error {
	return r.channel.ExchangeDeclare(
		exchangeName,
		kind,
		durable,
		autoDelete,
		false,
		false,
		nil,
	)
}

func (r *rabbitChannel) DeclareQueue(queueName string, args amqp.Table, durable, autoDelete bool) error {
	_, err := r.channel.QueueDeclare(
		queueName,
		durable,
		autoDelete,
		false,
		false,
		args,
	)
	return err
}

func (r *rabbitChannel) ConsumeQueue(queueName string, autoAck bool) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(
		queueName,
		r.uuid,
		autoAck,
		false,
		false,
		false,
		nil,
	)
}

func (r *rabbitChannel) BindQueue(queueName, key, exchange string, args amqp.Table) error {
	return r.channel.QueueBind(
		queueName,
		key,
		exchange,
		false,
		args,
	)
}
