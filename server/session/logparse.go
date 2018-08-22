package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// LogParser log过滤器
type LogParser struct{}

// Parse 解析logrus.Entry通过kafka hook转换的json
func (f LogParser) Parse(data []byte) ([]byte, error) {
	var e map[string]interface{}
	json.Unmarshal(data, &e)
	var host, l, msg, logTime string
	if v, ok := e["host"]; ok {
		host = v.(string)
		delete(e, "host")
	}
	if _, ok := e["topic"]; ok {
		delete(e, "topic")
	}
	if v, ok := e["level"]; ok {
		l = v.(string)
		delete(e, "level")
	}
	if v, ok := e["msg"]; ok {
		msg = v.(string)
		delete(e, "msg")
	}
	if v, ok := e["time"]; ok {
		logTime = v.(string)
		delete(e, "time")
	}

	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buffer := bytes.Buffer{}
	for _, k := range keys {
		buffer.WriteString(k)
		buffer.WriteString("=")

		switch v := e[k].(type) {
		case string:
			buffer.WriteString(v)
		case error:
			buffer.WriteString(v.Error())
		default:
			fmt.Fprint(&buffer, v)
		}
		buffer.WriteString(" ")
	}
	var level string
	switch l {
	case "info":
		level = "INFO"
	case "warning":
		level = "WARN"
	case "debug":
		level = "DEBU"
	case "error":
		level = "ERRO"
	case "panic":
		level = "PANI"
	case "fatal":
		level = "FATA"
	default:
		level = "UNKNOWN"
	}
	line := fmt.Sprintf("%s\t[%v]\t[%s]\t%-80s\t%s\n", level, logTime, host, msg, buffer.String())

	return []byte(line), nil
}
