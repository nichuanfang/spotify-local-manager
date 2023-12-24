package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nichuanfang/spotify-local-manager/util"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

var (
	//spotify客户端id
	spotifyClientID string
	//spotify客户端密钥
	spotifyClientSecret string
	//权限
	scopes []string
	//会话标识符
	state string
	//重定向URL
	redirectURL string
	//协程同步对象
	wg = &sync.WaitGroup{}
	//授权协程通道
	authChan = make(chan struct{})
	//服务停止信号
	stopChan = make(chan struct{})
	//认证器
	auth *spotifyauth.Authenticator
	//token文件所在目录
	tokenPath string
	//spotify本地文件所在目录
	spotifyLocalPath string
	//spotify本地临时文件(存放未分类mp3)所在目录
	spotifyLocalTempPath string
)

// 携带上下文的token
type tokenWithContext struct {
	// token
	token *oauth2.Token
	// 上下文
	ctx context.Context
}

func init() {
	spotifyClientID = os.Getenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")
	scopes = []string{
		spotifyauth.ScopeUserReadPrivate,
		spotifyauth.ScopeUserLibraryRead,
		spotifyauth.ScopeUserLibraryModify,
		spotifyauth.ScopePlaylistReadPrivate,
		spotifyauth.ScopePlaylistModifyPrivate,
		spotifyauth.ScopePlaylistModifyPublic,
	}
	if spotifyClientID == "" || spotifyClientSecret == "" {
		panic("spotify客户端ID和客户端密钥都需要设置!")
	}
	redirectURL = "http://127.0.0.1:9999/callback"
	state = util.GenerateRandString(10)
	//当前目录
	currDir, err := os.Getwd()
	if err == nil {
		tokenPath = filepath.Join(currDir, "token.json")
	} else {
		fmt.Errorf("获取当前目录错误")
		os.Exit(1)
	}
	spotifyLocalPath = "D:\\spotify\\spotify_local"
	spotifyLocalTempPath = "D:\\spotify\\spotify_local_temp"
}

// openAuthorizationURL 使用默认浏览器打开授权URL
func openAuthorizationURL() {
	authorizationURL := generateAuthorizationURL()
	//调用系统指令使用默认浏览器打开该URL
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", authorizationURL)
	err := cmd.Start()
	if err != nil {
		fmt.Errorf(err.Error())
	}
}

// 启动协程
func boot() {
	defer wg.Done()

	select {
	case <-authChan:
		fmt.Println("授权协程已准备好")
		//尝试打开存储token的json文件
		tokenFile, err := os.Open(tokenPath)
		if err == nil {
			defer tokenFile.Close()
			//token存在  则读取token.json 反序列化到内存中  不用OAuth2授权
			var token = new(oauth2.Token)
			decoder := json.NewDecoder(tokenFile)
			decoder.Decode(token)
			if err != nil {
				fmt.Println("无法解码token.json: ", err)
				os.Exit(1)
			}
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
			client := config.Client(ctx, token)
			sp := spotify.New(client)
			//直接进行业务处理
			success := handle(ctx, sp)
			if success {
				//终止callback协程
				stopChan <- struct{}{}
				break
			}
		} else {
			//1. 如果token.json不存在就发起授权的请求
			//		1.1发起打开授权URL的指令 最终会进入准备了回调接口的协程
			//     1.2 将获取到的token序列化到token.json中
			openAuthorizationURL()
		}
	}
}

// 授权协程
func callback(server *http.Server) {
	defer wg.Done()
	//创建路由对象
	router := gin.Default()

	// 注册路由
	router.GET("/callback", func(c *gin.Context) {
		//申请token
		token := exchangeCodeForToken(c.Writer, c.Request)
		//序列化token
		//os.Open()只能打开文件   os.Create()可以新建或覆写文件
		tokenFile, err := os.Create(tokenPath)
		if err != nil {
			fmt.Println("无法创建token.json文件")
			os.Exit(1)
		}
		defer tokenFile.Close()
		encoder := json.NewEncoder(tokenFile)
		err = encoder.Encode(token)
		if err != nil {
			fmt.Println("无法写入文件: ", err)
			os.Exit(1)
		}
		client := auth.Client(c, token)
		sp := spotify.New(client)
		handle(c, sp)
	})
	//绑定server
	server.Handler = router

	//启动server
	go func() {
		//server.ListenAndServe()会阻塞 直到发生错误
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("服务器启动失败: ", err)
		}
	}()

	// 服务器成功启动后延时1秒发送通知
	time.Sleep(1 * time.Second)
	fmt.Println("Auth服务成功启动!")
	//服务器成功启动 通知业务协程已准备就绪
	authChan <- struct{}{}

	//等待终止信号
	<-stopChan

	//关闭服务器
	if err := server.Shutdown(context.Background()); err != nil {
		fmt.Println("服务器关闭失败: ", err)
	}
}

func main() {

	wg.Add(2)

	server := &http.Server{
		Addr: ":9999",
	}
	// 启动协程
	go boot()
	// 认证协程
	go callback(server)

	// 等待两个协程执行完毕
	wg.Wait()
	//处理完成
	fmt.Println("处理完成!")
}
