package kafka

import (
	"errors"
	"net"
	"strconv"

	"github.com/go-kratos/kratos/v2/log"
	kafkaGo "github.com/segmentio/kafka-go"
)

func createConnection(addr string) (*kafkaGo.Conn, func()) {
	conn, err := kafkaGo.Dial("tcp", addr)
	if err != nil {
		log.Errorf("create kafka connection failed: %s", err.Error())
		return nil, nil
	}

	controller, err := conn.Controller()
	if err != nil {
		log.Errorf("create kafka controller failed: %s", err.Error())
	}

	controllerConn, err := kafkaGo.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}

	return controllerConn, func() {
		if err = conn.Close(); err != nil {
			log.Fatalf("failed to close kafka connection: %v", err)
		}
		if err = controllerConn.Close(); err != nil {
			log.Fatalf("failed to close kafka controller connection: %v", err)
		}
	}
}

func CreateTopic(addr string, topic string, numPartitions, replicationFactor int) error {
	conn, cleanFunc := createConnection(addr)
	if conn == nil {
		return errors.New("create kafka connection failed")
	}
	defer cleanFunc()

	err := conn.CreateTopics(kafkaGo.TopicConfig{
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	})
	if err != nil && errors.Is(err, kafkaGo.TopicAlreadyExists) {
		return nil
	}

	return err
}

func DeleteTopic(addr string, topics ...string) error {
	conn, cleanFunc := createConnection(addr)
	if conn == nil {
		return errors.New("create kafka connection failed")
	}
	defer cleanFunc()

	return conn.DeleteTopics(topics...)
}
