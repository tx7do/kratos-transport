package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateTopic(t *testing.T) {
	err := CreateTopic(defaultAddr, testTopic, 1, 1)
	assert.Nil(t, err)
}

func TestDeleteTopic(t *testing.T) {
	err := DeleteTopic(defaultAddr, testTopic)
	assert.Nil(t, err)
}
