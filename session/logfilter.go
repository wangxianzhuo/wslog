package session

import (
	"encoding/json"
	"fmt"
)

// LogFilter 日志过滤器
type LogFilter struct {
	FilterMap map[string]interface{}
}

// Filter ...
func (f LogFilter) Filter(data []byte, strategy FilterStrategy) (bool, error) {
	var msg map[string]interface{}

	err := json.Unmarshal(data, &msg)
	if err != nil {
		return false, fmt.Errorf("过滤消息%v异常: %v", string(data), err)
	}

	if len(f.FilterMap) < 1 {
		return true, nil
	}

	// s.logger.Debugf("data: %v", string(data))
	for k, v := range f.FilterMap {
		if value, ok := msg[k]; ok && (value == nil || v == value) {
			return true, nil
		}
	}
	return false, nil
}
