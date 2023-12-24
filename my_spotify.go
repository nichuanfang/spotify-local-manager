package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

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
		http.Error(w, "Could't get token", http.StatusInternalServerError)
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

// handle 业务处理方法
func handle(ctx context.Context, sp *spotify.Client) (success bool) {
	//search, err := sp.Search(ctx, "Drifting Soul", spotify.SearchTypeTrack)
	user, err := sp.CurrentUser(ctx)
	if err != nil {
		openAuthorizationURL()
		return
	}
	userId := user.ID
	fmt.Println(userId)

	//所有的tracks
	items, _ := sp.GetPlaylistItems(ctx, "3ojHX0ELdBa6VGgMwY2fYC")
	fmt.Println(items)

	//从items中读取所有本地文件元信息

	//将未分类的,分类错误的(以本地为准)本地文件移到spotify_local_temp文件夹

	//打开spotify客户端 本地来源关闭spotify_local 新增spotify_local_temp

	//分类完毕 再将本地来源改回去即可(关闭spotify_local_temp 新增spotify_local)

	success = true
	return
}
