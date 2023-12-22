package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/nichuanfang/spotify-local-manager/util"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var (
	//spotify客户端id
	spotifyClientId string
	//spotify客户端密钥
	spotifyClientSecret string
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
	//默认浏览器
	defaultBrowser string
	//浏览器是否打开
	browserState bool
	//认证器
	auth *spotifyauth.Authenticator
)

func init() {
	spotifyClientId = os.Getenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")
	if spotifyClientId == "" || spotifyClientSecret == "" {
		panic("spotify客户端ID和客户端密钥都需要设置!")
	}
	redirectURL = "http://127.0.0.1:9999/callback"
	state = GenerateRandString(10)
	//初始化浏览器参数
	initBrowser()
}

// 业务协程
func business(tokenChan chan string) {
	defer wg.Done()

loop:
	for {
		select {
		case <-authChan:
			fmt.Println("授权协程已准备好")
			//		发起打开授权URL的指令 最终会进入准备了回调接口的协程
			openAuthorizationURL()
		// 一直阻塞到获取到refreshToken
		case refreshToken := <-tokenChan:
			fmt.Println("开始业务处理...")
			fmt.Print(refreshToken)
			// 业务执行完毕 通知其他协程取消
			stopChan <- struct{}{}
			break loop

		}
	}
}

// 授权协程
func callback(server *http.Server, tokenChan chan string) {
	defer wg.Done()
	//创建路由对象
	router := gin.Default()

	// 注册路由
	router.GET("/callback", func(c *gin.Context) {
		//接收到回调之后 关闭浏览器
		if !browserState {
			//	如果之前是关闭状态 就关闭浏览器
			closeBrowser()
		}
		code := c.Query("code") //成功请求到token之后放入tokenChan
		fmt.Printf("开始根据code: %s 获取token\n", code)

		//通过code申请token

		tokenChan <- code + ":sddd8a9s8q234rhasdfhi7234"
	})
	//绑定server
	server.Handler = router

	//启动server
	go func() {
		//server.ListenAndServe()会阻塞 直到发生错误
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Errorf("服务器启动失败：%s\n", err)
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
		fmt.Errorf("服务器关闭失败: %s\n", err)
	}
}

func main() {

	wg.Add(2)

	// 创建一个token通道 值为refresh_token  业务协程拿到这个refresh_token之后 获取access_token来执行操作
	tokenChan := make(chan string)

	server := &http.Server{
		Addr: ":9999",
	}
	// 业务协程
	go business(tokenChan)
	// 认证协程
	go callback(server, tokenChan)

	// 等待两个协程执行完毕
	wg.Wait()
	//处理完成
	fmt.Println("处理完成!")
}
