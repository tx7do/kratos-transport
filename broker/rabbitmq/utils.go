package rabbitmq

import (
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"regexp"
)

var re = regexp.MustCompile("^amqp(s)?://.*")

func rabbitHeaderToMap(h amqp.Table) map[string]string {
	headers := make(map[string]string)
	for k, v := range h {
		headers[k], _ = v.(string)
	}
	return headers
}

func hasUrlPrefix(url string) bool {
	return re.MatchString(url)
}

func refitUrl(url string, enableTLS bool) string {
	if !hasUrlPrefix(url) {
		prefix := "amqp://"
		if enableTLS {
			prefix = "amqps://"
		}
		url = prefix + url
	}
	return url
}

func generateUUID() string {
	id, err := uuid.NewRandom()
	if err != nil {
		return ""
	}
	return id.String()
}
