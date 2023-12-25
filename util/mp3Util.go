package util

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/go-mp3"
)

// MP3MetaInfo mp3类
type MP3MetaInfo struct {
	//标题
	Title string
	//艺术家
	Artist string
	//专辑
	Album string
}

// ExtractMp3FromPath 根据路径解析MP3元信息
func ExtractMp3FromPath(mp3Path string) (mp3MetaInfo MP3MetaInfo, err error) {
	var mp3File *os.File
	mp3File, err = os.Open(mp3Path)
	if err != nil {
		fmt.Println("err: ", err)
		return
	}
	decoder, err := mp3.NewDecoder(mp3File)
	if err != nil {
		fmt.Println("err: ", err)
		return
	}
	decoder.Length()
	mp3File.Name()
	return
}
