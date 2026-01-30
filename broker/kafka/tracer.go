package kafka

import (
	"context"
	"strconv"

	kafkaGo "github.com/segmentio/kafka-go"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/tx7do/kratos-transport/tracing"
)

const (
	TracerMessageSystemKey = "kafka"
	SpanNameProducer       = "kafka-producer"
	SpanNameConsumer       = "kafka-consumer"
)

func (b *kafkaBroker) newProducerTracer() {
	b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, SpanNameProducer, b.options.Tracings...)
}

func (b *kafkaBroker) newConsumerTracer() {
	b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, SpanNameConsumer, b.options.Tracings...)
}

func (b *kafkaBroker) startProducerSpan(ctx context.Context, msg *kafkaGo.Message) (context.Context, trace.Span) {
	if b.producerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(TracerMessageSystemKey),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic),
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	if span != nil {
		otel.GetTextMapPropagator().Inject(ctx, carrier)
	}

	return ctx, span
}

func (b *kafkaBroker) finishProducerSpan(ctx context.Context, span trace.Span, partition int32, offset int64, err error) {
	if b.producerTracer == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingMessageIDKey.String(strconv.FormatInt(offset, 10)),
		semConv.MessagingKafkaPartitionKey.Int64(int64(partition)),
	}

	b.producerTracer.End(ctx, span, err, attrs...)
}

func (b *kafkaBroker) startConsumerSpan(ctx context.Context, msg *kafkaGo.Message) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(msg)

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(TracerMessageSystemKey),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(strconv.FormatInt(msg.Offset, 10)),
		semConv.MessagingKafkaPartitionKey.Int64(int64(msg.Partition)),
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *kafkaBroker) finishConsumerSpan(ctx context.Context, span trace.Span, err error) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(ctx, span, err)
}
