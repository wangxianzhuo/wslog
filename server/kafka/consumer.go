package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

// Consumer ...
type Consumer struct {
	Consumer          sarama.Consumer
	PartitionConsumer sarama.PartitionConsumer
	DataChan          chan []byte
	quit              chan struct{}
}

// New ...
func New(address []string, topic string, dataChan chan []byte) (*Consumer, error) {
	consumer, err := sarama.NewConsumer(address, nil)
	if err != nil {
		return nil, fmt.Errorf("新建到%v的topic为[%v]的kafka consumer 异常: %v", address, topic, err)
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		return nil, fmt.Errorf("新建topic为[%v]的kafka consumer partition 异常: %v", topic, err)
	}

	return &Consumer{
		Consumer:          consumer,
		PartitionConsumer: partitionConsumer,
		DataChan:          dataChan,
		quit:              make(chan struct{}),
	}, nil
}

// Start 启动监听kafka
func (c *Consumer) Start(logger *log.Entry) {
	logger.Debug("kafka consumer 启动监听")
	defer logger.Debug("kafka consumer 停止")
	for {
		select {
		case <-c.quit:
			close(c.quit)
			return
		case msg := <-c.PartitionConsumer.Messages():
			if msg == nil {
				logger.Warnf("获取空消息")
				continue
			}
			c.DataChan <- msg.Value
			// logger.Debugf("消费消息: %v", string(msg.Value))
		}
	}
}

// Close ...
func (c *Consumer) Close(logger *log.Entry) {
	if err := c.PartitionConsumer.Close(); err != nil {
		logger.Error(err)
	}
	if err := c.Consumer.Close(); err != nil {
		logger.Error(err)
	}
	c.quit <- struct{}{}
	close(c.DataChan)
}
