package rabbitmq

import "github.com/streadway/amqp"

func rabbitHeaderToMap(h amqp.Table) map[string]string {
	headers := make(map[string]string)
	for k, v := range h {
		headers[k], _ = v.(string)
	}
	return headers
}
