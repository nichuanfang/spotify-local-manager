package main

import (
	"fmt"
	"sort"
)

// pass	此为仿照python的pass方便调试 写的占位符函数 本身没有任何功能
func pass() {
}

func contains(slice []string, element string) bool {
	index := sort.Search(len(slice), func(i int) bool {
		return slice[i] >= element
	})
	return index < len(slice) && slice[index] == element
}

// handleError 错误处理函数
func handleError(err error) error {
	//造轮子
	if err != nil {
		fmt.Println("err:  ", err.Error())
	}
	return err
}

// specialHandleError  自定义逻辑处理错误
func specialHandleError(err error, logicFunc func(v ...interface{}) []interface{}) {
	if err != nil {
		fmt.Println("err: ", err)
	} else {
		//不报错执行自己的逻辑
		logicFunc()
	}
}
