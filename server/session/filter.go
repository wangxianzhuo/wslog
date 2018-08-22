package session

// Filter 过滤器接口
type Filter interface {
	Filter(data []byte, strategy FilterStrategy) (bool, error)
}

// FilterStrategy 过滤策略
type FilterStrategy struct{}
