package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	
	"github.com/nichuanfang/spotify-local-manager/util"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

// 存储歌单名与ID的映射
var playListMap = make(map[string]spotify.ID)

// 生成授权URL
func generateAuthorizationURL() (authorizationURL string) {
	//生成授权URL
	//认证器初始化
	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURL),
		spotifyauth.WithClientID(spotifyClientID),
		spotifyauth.WithClientSecret(spotifyClientSecret),
		spotifyauth.WithScopes(scopes...))
	authorizationURL = auth.AuthURL(state)
	return
}

// 通过code交换token
func exchangeCodeForToken(w gin.ResponseWriter, r *http.Request) *oauth2.Token {
	token, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Could't get Token", http.StatusInternalServerError)
		return nil
	}
	// 成功获取token后的关闭标签页
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	//反引号` 是字符串的原始格式  换行无需使用转义字符 很方便
	//[]byte() 可以将任意类型转为字节切片
	w.Write([]byte(`
		<script>
		window.close();
		</script>
		`))
	//大部分情况不需要手动调用w.Flush()将缓冲区数据发送给客户端并关闭连接,因为ResponseWriter会自动调用;如果需要立即在当前位置立即将缓冲区数据发送给客户端且关闭连接需要手动调用w.Flush()方法
	w.Flush()
	return token
}

// getClient 通过token获取spotify客户端指针对象
func getClient(r *http.Request, token *oauth2.Token) *http.Client {
	return auth.Client(r.Context(), token)
}

//获取本地文件夹`spotifyLocalPath`的歌单元信息

// 数据结构:  key: 歌单名称 string   value:  歌曲名称切片 []util.MP3
func getLocalMusicMetaData() map[string][]util.MP3MetaInfo {
	//初始化一个映射
	res := make(map[string][]util.MP3MetaInfo)
	//读取spotifyLocalPath
	filepath.Walk(spotifyLocalPath, func(path string, info fs.FileInfo, err error) error {
		if info != nil && info.IsDir() && info.Name() != "spotify_local" {
			res[info.Name()] = make([]util.MP3MetaInfo, 0)
		} else if strings.HasSuffix(info.Name(), ".mp3") {
			mp3, err := util.ExtractMp3FromPath(path)
			if err != nil {
				//当前mp3无法处理 直接跳过
				return nil
			}
			tracks, ok := res[mp3.PlayListName]
			if ok {
				//如果存在key
				tracks := append(tracks, mp3)
				res[mp3.PlayListName] = tracks
			}
		}
		return handleError(err)
	})

	return res
}

// 加载临时文件夹到序列化数据中
func loadLocalTempMusic() map[string][]util.MP3MetaInfo {
	//初始化一个映射
	res := make(map[string][]util.MP3MetaInfo)
	//读取spotifyLocalPath
	filepath.Walk(spotifyLocalTempPath, func(path string, info fs.FileInfo, err error) error {
		if info != nil && info.IsDir() && info.Name() != "spotify_local_temp" {
			res[info.Name()] = make([]util.MP3MetaInfo, 0)
		} else if strings.HasSuffix(info.Name(), ".mp3") {
			mp3, err := util.ExtractMp3FromPath(path)
			if err != nil {
				//当前mp3无法处理 直接跳过
				return nil
			}
			tracks, ok := res[mp3.PlayListName]
			if ok {
				//如果存在key
				tracks := append(tracks, mp3)
				res[mp3.PlayListName] = tracks
			}
		}
		return handleError(err)
	})
	// 移除切片长度为0的键值对
	for key, value := range res {
		if len(value) == 0 {
			delete(res, key)
		}
	}
	return res
}

// getAllPlayLists 获取所有的歌单
func getAllPlayLists(sp *spotify.Client, ctx context.Context, userId string) []spotify.SimplePlaylist {
	playlistsForUser, err := sp.GetPlaylistsForUser(ctx, userId, spotify.Limit(50))
	if err != nil {
		fmt.Println("歌单查询失败: ", err)
		os.Exit(1)
	}
	total := playlistsForUser.Total
	if total == 0 {
		return make([]spotify.SimplePlaylist, 0)
	}
	//每页的数量
	limit := playlistsForUser.Limit
	//每页的初始偏移量
	offset := limit
	playlists := playlistsForUser.Playlists
	//循环获取歌单
	for offset < total {
		getPlaylistsForUser, err := sp.GetPlaylistsForUser(ctx, userId, spotify.Limit(limit), spotify.Offset(offset))
		currPlaylists := getPlaylistsForUser.Playlists
		if err != nil {
			fmt.Println("查询歌单失败: ", err)
			os.Exit(1)
		} else if len(currPlaylists) == 0 {
			break
		}
		playlists = append(playlists, currPlaylists...)
		//每一轮循环偏移量增加
		offset += limit
	}

	return playlists
}

// getAllPlayListsIds 获取所有的歌单的id和name
func getAllPlayListsIds(sp *spotify.Client, ctx context.Context, userId string) []map[string]string {
	lists := getAllPlayLists(sp, ctx, userId)
	if len(lists) == 0 {
		//返回的是映射集合
		return make([]map[string]string, 0)
	}
	// [!NOTE]
	//切片底层是数组 为了避免扩容影响性能 需要指定一个初始容量
	//映射底层是哈希表 容量是动态增加的 不是扩容 所以可以不指定容量
	res := make([]map[string]string, 0)
	for _, item := range lists {
		playListMap := make(map[string]string)
		playListMap["name"] = item.Name
		playListMap["id"] = item.ID.String()
		res = append(res, playListMap)
	}
	return res
}

// getTracksByPlayList 根据歌单 获取歌单所有的本地曲目
func getTracksByPlayList(sp *spotify.Client, ctx context.Context, playList spotify.SimplePlaylist) ([]util.MP3MetaInfo, error) {
	pageItems, err := sp.GetPlaylistItems(ctx, playList.ID, spotify.Limit(100))
	if err != nil {
		fmt.Println("err: ", err)
		return make([]util.MP3MetaInfo, 0), err
	} else if pageItems.Total == 0 {
		return make([]util.MP3MetaInfo, 0), nil
	}
	limit := pageItems.Limit
	offset := limit
	total := pageItems.Total
	items := pageItems.Items
	//创建一个装载本地曲目的切片
	localTracks := make([]util.MP3MetaInfo, 0)
	//初始化装载
	for _, item := range items {
		if item.IsLocal {
			trackName := item.Track.Track.Name
			artists := item.Track.Track.Artists
			if len(artists) == 0 || artists[0].Name == "" {
				continue
			}
			trackArtist := artists[0].Name
			trackAlbum := item.Track.Track.Album.Name
			localTracks = append(localTracks, util.MP3MetaInfo{
				Title:        trackName,
				Artist:       trackArtist,
				Album:        trackAlbum,
				PlayListName: playList.Name,
			})
		}
	}
	//更新items
	for offset < total {
		playlistItems, err := sp.GetPlaylistItems(ctx, playList.ID, spotify.Limit(limit), spotify.Offset(offset))
		if err != nil || playlistItems.Total == 0 {
			break
		}
		for _, item := range playlistItems.Items {
			if item.IsLocal {
				trackName := item.Track.Track.Name
				artists := item.Track.Track.Artists
				if len(artists) == 0 || artists[0].Name == "" {
					continue
				}
				trackArtist := artists[0].Name
				trackAlbum := item.Track.Track.Album.Name
				localTracks = append(localTracks, util.MP3MetaInfo{
					Title:        trackName,
					Artist:       trackArtist,
					Album:        trackAlbum,
					PlayListName: playList.Name,
				})
			}
		}
		//更新offset
		offset += limit
	}
	return localTracks, nil
}

// isTrackInLocalTracks 判断spotify已收录元信息的曲目是否存在于本地库
func isTrackInLocalTracks(track util.MP3MetaInfo, localTracks []util.MP3MetaInfo) (flag bool, filename string) {

	//	hahaha
loop:
	for _, localTrack := range localTracks {
		if util.EvaluateSimilar(localTrack.Artist, track.Artist) &&
			util.EvaluateSimilar(localTrack.Title, track.Title) &&
			util.EvaluateSimilar(localTrack.Album, track.Album) {
			if localTrack.FileName != "" {
				filename = localTrack.FileName
			} else if track.FileName != "" {
				filename = track.FileName
			}
			flag = true
			break loop
		}
	}
	return
}

// removeTrack 移除曲目
func removeTrack(localTracks []util.MP3MetaInfo, track util.MP3MetaInfo) []util.MP3MetaInfo {
	newTracks := make([]util.MP3MetaInfo, 0)
	for _, localTrack := range localTracks {
		if util.EvaluateSimilar(localTrack.Artist, track.Artist) &&
			util.EvaluateSimilar(localTrack.Title, track.Title) &&
			util.EvaluateSimilar(localTrack.Album, track.Album) {
			continue
		}
		newTracks = append(newTracks, localTrack)
	}
	return newTracks
}

// diffTracks 比较本地曲目和线上本地曲目 过滤出未分类和分类错误的曲目
func diffTracks(localTracks []util.MP3MetaInfo, tracks []util.MP3MetaInfo) ([]util.MP3MetaInfo, []util.MP3MetaInfo) {
	//所有标准皆以本地为准
	//如果tracks中的曲目 在localTracks中不存在  说明该文件属于分类错误 将这些文件过滤出来
	//localTracks-tracks剩余的曲目是需要分类的
	// 在spotifyLocalTemp文件夹创建歌单分类文件夹 将过滤出的这些曲目移动过去
	tickedTracks := make([]util.MP3MetaInfo, 0)
	for _, track := range tracks {
		if flag, filename := isTrackInLocalTracks(track, localTracks); flag {
			track.FileName = filename
			//从localTracks中移除该曲目
			localTracks = removeTrack(localTracks, track)
			tickedTracks = append(tickedTracks, track)
		}
	}
	return localTracks, tickedTracks
}

func moveToTemp(unHandledTracks []util.MP3MetaInfo, playListName string) {
	basePath := filepath.Join(spotifyLocalPath, playListName)
	tempBasePath := filepath.Join(spotifyLocalTempPath, playListName)
	mp3Files := make([]util.MP3MetaInfo, 0)
	err := filepath.Walk(tempBasePath, func(path string, info fs.FileInfo, err error) error {
		//如果当前文件是mp3
		if err != nil && info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".mp3") {
			metaInfo, err := util.ExtractMp3FromPath(path)
			if err != nil {
				//当前mp3处理失败下一个
				return nil
			}
			mp3Files = append(mp3Files, metaInfo)
		}
		return err
	})
	if err != nil {
		//	路径不存在 创建目录
		err = os.Mkdir(tempBasePath, os.ModeDir)
		if err != nil {
			fmt.Println("创建目录失败")
			return
		}
		moveToTemp(unHandledTracks, playListName)
		return
	}
	//	遍历unHandledTracks 如果存在和mp3Files中匹配的mp3文件就跳过
	for _, track := range unHandledTracks {
		if flag, _ := isTrackInLocalTracks(track, mp3Files); flag {
			continue
		}
		source := filepath.Join(basePath, track.FileName)
		dest := filepath.Join(tempBasePath, track.FileName)
		//移动到对应的临时文件夹
		err := os.Rename(source, dest)
		if err != nil {
			_ = closeSpotifyProcess()
			err = os.Rename(source, dest)
			if err != nil {
				fmt.Println("文件移动失败: ", err)
				continue
			}
		}
	}
}

// 移动至本地文件夹
func moveToLocal(tickedTracks []util.MP3MetaInfo, playListName string) {
	basePath := filepath.Join(spotifyLocalPath, playListName)
	tempBasePath := filepath.Join(spotifyLocalTempPath, playListName)
	mp3Files := make([]util.MP3MetaInfo, 0)
	filepath.Walk(basePath, func(path string, info fs.FileInfo, err error) error {
		//如果当前文件是mp3
		if err != nil && info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".mp3") {
			metaInfo, err := util.ExtractMp3FromPath(path)
			if err != nil {
				//当前mp3处理失败下一个
				return nil
			}
			mp3Files = append(mp3Files, metaInfo)
		}
		return err
	})
	//	遍历unHandledTracks 如果存在和mp3Files中匹配的mp3文件就跳过
	for _, track := range tickedTracks {
		if flag, _ := isTrackInLocalTracks(track, mp3Files); flag {
			continue
		}
		//移动到对应的临时文件夹
		err := os.Rename(filepath.Join(tempBasePath, track.FileName), filepath.Join(basePath, track.FileName))
		if err != nil {
			_ = closeSpotifyProcess()
			err := os.Rename(filepath.Join(tempBasePath, track.FileName), filepath.Join(basePath, track.FileName))
			if err != nil {
				fmt.Println("文件移动失败: ", err)
				continue
			}
		}
	}
}

// 关闭spotify进程
func closeSpotifyProcess() error {
	cmd := exec.Command("taskkill", "/IM", "Spotify.exe", "/F")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("关闭 Spotify 进程失败：%v", err)
	}
	return nil
}

// 获取进程的详细信息
func getProcessInfo(processName string) (string, error) {
	cmd := exec.Command("wmic", "process", "where", "name='"+processName+"'", "get", "ExecutablePath", "/format:list")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// 从进程详细信息中提取文件路径
func extractFilePath(processInfo string) string {
	lines := strings.Split(processInfo, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ExecutablePath=") {
			return strings.TrimPrefix(line, "ExecutablePath=")
		}
	}
	return ""
}

// handle 业务处理方法
func handle(ctx context.Context, sp *spotify.Client) (success bool) {
	fmt.Println("处理中...")
	//search, err := sp.Search(ctx, "Drifting Soul", spotify.SearchTypeTrack)
	user, err := sp.CurrentUser(ctx)
	if err != nil {
		openAuthorizationURL()
		return
	}
	userId := user.ID
	//获取所有的playlists
	playLists := getAllPlayLists(sp, ctx, userId)
	//make是返回已经初始化好的对象 不过只能针对于[切片 映射 通道这三种类型] 适用于内置类型
	//new是返回未初始化的对象0值指针 为对象分配0值内存 但是还未初始化 还是nil 针对自定义类型 (new返回对象的指针 但是该对象还是nil 未初始化)
	//var tracks =  make([]spotify.SimpleTrack,10)
	if len(playLists) == 0 {
		//说明歌单是空的
		success = true
		return
	}
	for _, list := range playLists {
		playListMap[list.Name] = list.ID
	}

	//获取本地元数据
	localMusicMetaData := getLocalMusicMetaData()
	//读取临时文件夹 放到serializeData中
	serializeData := loadLocalTempMusic()

	//遍历歌单集合 过滤出本地  `未分类`  和   `分类错误的歌曲(以本地为准) 即能在本地文件夹找到 同时该mp3文件所属父文件夹的名称与当前歌单名称不一致`
	for _, playList := range playLists {
		//查询本地元数据 通过key = 歌单名称查询 是否在映射中存在
		localTracks, ok := localMusicMetaData[playList.Name]
		if ok {
			//	key存在!
			//根据playListId查询在线歌单的tracks
			tracks, err := getTracksByPlayList(sp, ctx, playList)
			if err != nil {
				//如果获取歌单失败 处理下一个歌单
				continue
			}
			//处理本地曲目localTracks和在线本地曲目tracks 过滤出满足条件的曲目路径集合
			//将未分类的,分类错误的(以本地为准)本地文件移到spotify_local_temp文件夹
			//打开spotify客户端 本地来源关闭spotify_local 新增spotify_local_temp
			//分类完毕 再将本地来源改回去即可(关闭spotify_local_temp 新增spotify_local)

			//if  len(tracks) == 0{
			//	//本地的曲目需要全部同步过去
			//	continue
			//}
			//if len(localTracks) == 0 {
			//	// 说明spotify服务器以前同步了元数据 但是本地文件丢失 需要手动将服务器这部分文件元数据删除
			//	continue
			//}
			unHandledTracks, _ := diffTracks(localTracks, tracks)
			if len(unHandledTracks) != 0 {
				//移动到temp文件夹
				moveToTemp(unHandledTracks, playList.Name)
				//如果serializeData存在歌单key 则选择加入
				if data, ok := serializeData[playList.Name]; ok {
					serializeData[playList.Name] = append(data, unHandledTracks...)
				} else {
					serializeData[playList.Name] = unHandledTracks
				}
			}
		} else {
			//本地音乐库不存在该歌单 创建该歌单文件夹
			err := os.Mkdir(filepath.Join(spotifyLocalPath, playList.Name), 0755)
			if err != nil {
				fmt.Printf("歌单: %v创建失败: %v", playList.Name, err)
			} else {
				fmt.Printf("已创建本地歌单: %v", playList.Name)
			}
		}
	}
	uncateforizedFile, err := os.Create(filepath.Join(spotifyConfigBasePath, "uncategorized.json"))
	if err != nil {
		return false
	}
	defer uncateforizedFile.Close()
	//序列化成json到当前文件夹
	encoder := json.NewEncoder(uncateforizedFile)
	err = encoder.Encode(serializeData)
	if err != nil {
		fmt.Println("序列化数据失败: ", err)
		return false
	}
	//打印曲目分类信息 [歌单名称]   [起始序号]   [终止序号]
	success = true
	return
}

func getCategorizeStat(uncategorizedData map[string][]util.MP3MetaInfo, leftTracksChan chan map[string][]util.MP3MetaInfo, tickedTracksFilesChan chan []map[string]string, exitSignal chan struct{}) {
	//创建uncategorizedData的深拷贝对象
	copyUncategorizedData := make(map[string][]util.MP3MetaInfo)
	for k, v := range uncategorizedData {
		copyUncategorizedData[k] = v
	}
	tickedTracksData := make([]map[string]string, 0)
	ctx := context.Background()
	config := &oauth2.Config{
		ClientID:     spotifyClientID,
		ClientSecret: spotifyClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  spotifyauth.AuthURL,
			TokenURL: spotifyauth.TokenURL,
		},
	}
	token := <-tokenChan
	client := config.Client(ctx, token)
	sp := spotify.New(client)

	for {
		//每完成一个歌单的分类 就减少一个歌单的查询
		newData := make(map[string][]util.MP3MetaInfo)
		//遍历uncategorizedData临时文件夹
		for playListName, localTracks := range copyUncategorizedData {
			//根据歌单名称 在映射表里查询对应的歌单ID
			playlistID, ok := playListMap[playListName]
			if !ok || playlistID == "" {
				//不存在这样的歌单或者id为空
				continue
			}
			//根据歌单ID 查询spotify在线元数据 得到本地曲目元数据切片
			tracks, err := getTracksByPlayList(sp, ctx, spotify.SimplePlaylist{ID: playlistID, Name: playListName})
			if err != nil {
				//查询歌单曲目失败 可能是受到了rate limit
				fmt.Println("查询歌单曲目元信息失败: ", err)
				os.Exit(1)
			}
			//已剔除的曲目

			leftTracks, tickedTracks := diffTracks(localTracks, tracks)
			if len(leftTracks) != 0 {
				newData[playListName] = leftTracks
			} else {
				//	此歌单处理完毕
				delete(copyUncategorizedData, playListName)
			}

			// 每剔除一首 就移动一首
			if len(tickedTracks) != 0 {
				for _, track := range tickedTracks {
					tickedTracksData = append(tickedTracksData, map[string]string{
						"source": filepath.Join(spotifyLocalTempPath, playListName, track.FileName),
						"dest":   filepath.Join(spotifyLocalPath, playListName, track.FileName),
					})
				}
			}
		}
		if len(newData) == 0 {
			fmt.Println("分类已完成!")
			go func() {
				leftTracksChan <- newData
				tickedTracksFilesChan <- tickedTracksData
				exitSignal <- struct{}{}
				tokenChan <- token
			}()
			break
		}
		leftTracksChan <- newData
		time.Sleep(5 * time.Second)
	}

}

// 移动文件
func postProcess(tickedTracksFilesChan chan []map[string]string) {
	data := <-tickedTracksFilesChan
	sourceRecover := make([]string, 0)
	for _, item := range data {
		source := item["source"]
		if contains(sourceRecover, source) {
			continue
		} else {
			sourceRecover = append(sourceRecover, source)
		}
		dest := item["dest"]
		err := os.Rename(source, dest)
		if err != nil {
			needSpotifyRecover = true
			_ = closeSpotifyProcess()
			err = os.Rename(source, dest)
			if err != nil {
				fmt.Println("移动文件失败: ", err)
				os.Exit(1)
			}
		}
	}
	if needSpotifyRecover {
		// 获取 Spotify 进程的详细信息
		if spotifyAppPath == "" {
			fmt.Println("Spotify.exe process is not found")
			return
		}
		cmd := exec.Command(spotifyAppPath)
		err := cmd.Start()
		if err != nil {
			fmt.Println("打开Spotify失败: ", err)
			return
		}
	}

}
