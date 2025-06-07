package kafka

import (
	"errors"
	"net"
	"strconv"

	kafkaGo "github.com/segmentio/kafka-go"
)

func createConnection(addr string) (*kafkaGo.Conn, func()) {
	conn, err := kafkaGo.Dial("tcp", addr)
	if err != nil {
		LogErrorf("create kafka connection failed: %s", err.Error())
		return nil, nil
	}

	controller, err := conn.Controller()
	if err != nil {
		LogErrorf("create kafka controller failed: %s", err.Error())
		return nil, nil
	}

	controllerConn, err := kafkaGo.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}

	return controllerConn, func() {
		if err = conn.Close(); err != nil {
			LogFatalf("failed to close kafka connection: %v", err)
		}
		if err = controllerConn.Close(); err != nil {
			LogFatalf("failed to close kafka controller connection: %v", err)
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
