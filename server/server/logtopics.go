package server

import (
	"fmt"

	"github.com/Shopify/sarama"
)

// GetTopics 获取连接到的kafka的topic
func GetTopics(address []string) ([]string, error) {
	consumer, err := sarama.NewConsumer(address, nil)
	if err != nil {
		return nil, fmt.Errorf("新建到kafka[%v]的连接异常: %v", address, err)
	}
	defer consumer.Close()

	return consumer.Topics()
}
