package util

import (
	"fmt"
	"net"
)

// IsPortInUse 检测端口是否被占用
func IsPortInUse(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		// 端口被占用或发生其他错误
		return true
	}
	defer listener.Close()
	return false
}
