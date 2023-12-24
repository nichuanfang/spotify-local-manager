package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

// 生成授权URL
func generateAuthorizationURL() (authorizationURL string) {
	//生成授权URL
	authScopes := []string{spotifyauth.ScopeUserReadPrivate}
	//认证器初始化
	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURL),
		spotifyauth.WithClientID(spotifyClientId),
		spotifyauth.WithClientSecret(spotifyClientSecret),
		spotifyauth.WithScopes(authScopes...))
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
