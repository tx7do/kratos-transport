package kafka

import (
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
)

type WriterConfig struct {
	// The balancer used to distribute messages across partitions.
	//
	// The default is to use a round-robin distribution.
	Balancer kafkaGo.Balancer

	// Limit on how many attempts will be made to deliver a message.
	//
	// The default is to try at most 10 times.
	MaxAttempts int

	// Limit on how many messages will be buffered before being sent to a
	// partition.
	//
	// The default is to use a target batch size of 100 messages.
	BatchSize int

	// Limit the maximum size of a request in bytes before being sent to
	// a partition.
	//
	// The default is to use a kafka default value of 1048576.
	BatchBytes int64

	// Time limit on how often incomplete message batches will be flushed to
	// kafka.
	//
	// The default is to flush at least every second.
	BatchTimeout time.Duration

	// Timeout for read operations performed by the Writer.
	//
	// Defaults to 10 seconds.
	ReadTimeout time.Duration

	// Timeout for write operation performed by the Writer.
	//
	// Defaults to 10 seconds.
	WriteTimeout time.Duration

	// Number of acknowledges from partition replicas required before receiving
	// a response to a produce request. The default is -1, which means to wait for
	// all replicas, and a value above 0 is required to indicate how many replicas
	// should acknowledge a message to be considered successful.
	//
	// This version of kafka-go (v0.3) does not support 0 required acks, due to
	// some internal complexity implementing this with the Kafka protocol. If you
	// need that functionality specifically, you'll need to upgrade to v0.4.
	RequiredAcks kafkaGo.RequiredAcks

	// Setting this flag to true causes the WriteMessages method to never block.
	// It also means that errors are ignored since the caller will not receive
	// the returned value. Use this only if you don't care about guarantees of
	// whether the messages were written to kafka.
	Async bool

	// If not nil, specifies a logger used to report internal changes within the
	// writer.
	Logger kafkaGo.Logger

	// ErrorLogger is the logger used to report errors. If nil, the writer falls
	// back to using Logger instead.
	ErrorLogger kafkaGo.Logger

	// AllowAutoTopicCreation notifies writer to create topic if missing.
	AllowAutoTopicCreation bool
}
