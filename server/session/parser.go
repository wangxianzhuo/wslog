package session

// Parser 解析器接口
type Parser interface {
	Parse(data []byte) ([]byte, error)
}
