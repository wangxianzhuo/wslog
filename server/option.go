package server

// KafkaOpt log所在kafka连接信息
type KafkaOpt struct {
	Brokers []string
	Topic   string
}

// FilterOpt 过滤配置
type FilterOpt struct {
	Key   string
	Value string
}
