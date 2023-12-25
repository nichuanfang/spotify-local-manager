package util

import (
	"fmt"
	"path/filepath"

	"github.com/bogem/id3v2"
)

// MP3MetaInfo mp3类
type MP3MetaInfo struct {
	//标题
	Title string
	//艺术家
	Artist string
	//专辑
	Album string
	//所属歌单名称
	PlayListName string
	//文件名称
	FileName string
}

// ExtractMp3FromPath 根据路径解析MP3元信息
func ExtractMp3FromPath(mp3Path string) (MP3MetaInfo, error) {
	mp3Tag, err := id3v2.Open(mp3Path, id3v2.Options{
		Parse: true,
	})
	if err != nil {
		fmt.Println("err: ", err)
		return MP3MetaInfo{}, err
	}
	parentDirPath, fileName := filepath.Split(mp3Path)
	return MP3MetaInfo{
		Title:        mp3Tag.Title(),
		Artist:       mp3Tag.Artist(),
		Album:        mp3Tag.Album(),
		PlayListName: filepath.Base(parentDirPath),
		FileName:     fileName,
	}, nil
}
