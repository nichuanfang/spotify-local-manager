package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	//gin本地监听端口 默认9999
	listenPort int
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
	//项目配置根目录
	spotifyConfigBasePath string
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

// spotifyPrincipal spotify凭证信息
type spotifyPrincipal struct {

	// [!Import]
	// 在序列化和反序列化时 只有首字母大写的字段才可以被序列化和反序列化

	//获取到的 OAuth Token
	Token *oauth2.Token
	//客户端ID
	SpotifyClientID string
	//客户端密钥
	SpotifyClientSecret string
	//监听的端口 对应回调URL
	Port int
}

// 返回重定向URL
func (principal *spotifyPrincipal) getRedirectURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", principal.Port)
}

func init() {
	scopes = []string{
		spotifyauth.ScopeUserReadPrivate,
		spotifyauth.ScopeUserLibraryRead,
		spotifyauth.ScopeUserLibraryModify,
		spotifyauth.ScopePlaylistReadPrivate,
		spotifyauth.ScopePlaylistModifyPrivate,
		spotifyauth.ScopePlaylistModifyPublic,
	}
	state = util.GenerateRandString(10)
	//home目录
	homeDir, err := os.UserHomeDir()
	if err == nil {
		spotifyConfigBasePath = filepath.Join(homeDir, ".spotifyLocalManager")
		err := os.MkdirAll(spotifyConfigBasePath, os.ModeDir)
		if err != nil {
			fmt.Println("无法创建目录:", err)
			os.Exit(1)
		}
		tokenPath = filepath.Join(spotifyConfigBasePath, "Token.json")

	} else {
		fmt.Println("获取用户目录错误")
		os.Exit(1)
	}
	//根据执行exe的目录来推断spotify_local和spotify_local_temp
	currDir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取当前目录失败: ", err)
		os.Exit(1)
	}
	//如果当前文件夹有go.mod文件 说明是开发环境 currDir根据实际目录修改
	_, err = os.Stat(filepath.Join(currDir, "go.mod"))
	if err == nil {
		//存在go.mod文件 说明是本地开发环境
		currDir = "D:\\spotify"
	}
	//处理spotify本地文件夹
	spotifyLocalPath = filepath.Join(currDir, "spotify_local")
	//如果该文件夹不存在 创建一个
	_, err = os.Stat(spotifyLocalPath)
	if os.IsNotExist(err) {
		//	文件夹不存在 创建一个
		createErr := os.MkdirAll(spotifyLocalPath, os.ModeDir)
		if createErr != nil {
			fmt.Println("创建文件夹失败! ", err)
			os.Exit(1)
		}
		fmt.Println("成功创建spotify本地文件夹!")
	} else if err != nil {
		fmt.Println("检查文件夹失败: ", err)
		os.Exit(1)
	}
	//处理spotify临时文件夹
	spotifyLocalTempPath = filepath.Join(currDir, "spotify_local_temp")
	//如果临时文件夹不存在 则创建
	_ = os.RemoveAll(spotifyLocalTempPath)
	_ = os.MkdirAll(spotifyLocalTempPath, os.ModeDir)
}

// openAuthorizationURL 使用默认浏览器打开授权URL
func openAuthorizationURL() {
	fmt.Printf("请去 https://developer.spotify.com/dashboard 设置里添加回调地址: %s\n", redirectURL)
	authorizationURL := generateAuthorizationURL()
	//调用系统指令使用默认浏览器打开该URL
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", authorizationURL)
	err := cmd.Start()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func initOauthConfig(clientID string, clientSecret string, port int) {
	//如果tokenPath不存在 就要求用户输入这两个值 ; 如果存在 在反序列号token.json成功之后 将对应的值设置到客户端ID,密钥,端口中
	reader := bufio.NewReader(os.Stdin)
	if clientID != "" {
		spotifyClientID = clientID
	} else {
		// 获取 Spotify 客户端 ID
		for {
			fmt.Print("请输入 Spotify 客户端 ID：")
			clientID, _ := reader.ReadString('\n')
			if strings.TrimSpace(clientID) == "" {
				fmt.Println("客户端ID必填!")
				continue
			}
			spotifyClientID = strings.TrimSpace(clientID)
			break
		}
	}
	if clientSecret != "" {
		spotifyClientSecret = clientSecret
	} else {
		for {
			// 获取 Spotify 客户端密钥
			fmt.Print("请输入 Spotify 客户端密钥：")
			clientSecret, _ := reader.ReadString('\n')
			if strings.TrimSpace(clientSecret) == "" {
				fmt.Println("客户端密钥必填!")
				continue
			}
			spotifyClientSecret = strings.TrimSpace(clientSecret)
			break
		}
	}
	if port != 0 {
		listenPort = port
		redirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", listenPort)
	} else {
		// 获取本地监听端口
		for {
			fmt.Print("请输入本地监听端口（默认为 9999）：")
			port, _ := reader.ReadString('\n')
			inputPort := strings.TrimSpace(port)
			if inputPort == "" {
				listenPort = 9999
				redirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", listenPort)
				break
			}
			listenPort, err := strconv.Atoi(inputPort)
			if err != nil {
				fmt.Println("端口必须为整数!")
				continue
			} else if util.IsPortInUse(listenPort) {
				fmt.Printf("端口: %v已被占用,请更换!", listenPort)
			}
			redirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", listenPort)
			break
		}
	}
	openAuthorizationURL()
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
			principal := new(spotifyPrincipal)
			decoder := json.NewDecoder(tokenFile)
			err := decoder.Decode(principal)
			if err != nil {
				fmt.Println("无法解码token.json: ", err)
				os.Exit(1)
			} else if principal.Token == nil {
				if principal.SpotifyClientID != "" {
					initOauthConfig(principal.SpotifyClientID, principal.SpotifyClientSecret, principal.Port)
				} else {
					initOauthConfig(spotifyClientID, spotifyClientSecret, listenPort)
				}
				break
			}
			spotifyClientID = principal.SpotifyClientID
			spotifyClientSecret = principal.SpotifyClientSecret
			listenPort = principal.Port
			redirectURL = principal.getRedirectURL()
			ctx := context.Background()
			config := &oauth2.Config{
				ClientID:     principal.SpotifyClientID,
				ClientSecret: principal.SpotifyClientSecret,
				RedirectURL:  principal.getRedirectURL(),
				Scopes:       scopes,
				Endpoint: oauth2.Endpoint{
					AuthURL:  spotifyauth.AuthURL,
					TokenURL: spotifyauth.TokenURL,
				},
			}
			client := config.Client(ctx, principal.Token)
			sp := spotify.New(client)
			//直接进行业务处理
			success := handle(ctx, sp)
			if success {
				//终止callback协程
				stopChan <- struct{}{}
				break
			}
		} else {
			//. Token.json不存在
			initOauthConfig(spotifyClientID, spotifyClientSecret, listenPort)
			break
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
		encoder := json.NewEncoder(tokenFile)

		err = encoder.Encode(spotifyPrincipal{
			Token:               token,
			SpotifyClientID:     spotifyClientID,
			SpotifyClientSecret: spotifyClientSecret,
			Port:                listenPort,
		})
		//err = encoder.Encode(Token)
		if err != nil {
			fmt.Println("无法写入文件: ", err)
			os.Exit(1)
		}
		err = tokenFile.Close()
		if err != nil {
			fmt.Println("无法关闭文件: ", err)
			os.Exit(1)
		}
		client := auth.Client(c, token)
		sp := spotify.New(client)
		success := handle(c, sp)
		if success {
			stopChan <- struct{}{}
		}
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
	//数据处理完成

	//如果生成的uncategorized.json不是空的json串 则开启一个服务 去提供访问
	uncategorizedFile, err := os.Open(filepath.Join(spotifyConfigBasePath, "uncategorized.json"))
	if os.IsNotExist(err) {
		fmt.Println("处理完成! \n3秒后关闭此窗口...")
		time.Sleep(3 * time.Second)
		return
	} else if err == nil {
		uncategorizedData := make(map[string][]map[string]string)
		//	对uncategorizedFile进行反序列化 如果是个空结果 说明没有待分类的曲目;如果不是空 取出结果 开启server 向用户提供端点 使用默认浏览器打开该URL
		decoder := json.NewDecoder(uncategorizedFile)
		err := decoder.Decode(&uncategorizedData)
		if err != nil {
			fmt.Println("反序列化失败! ", err)
			os.Exit(1)
		}
		if uncategorizedData == nil || len(uncategorizedData) == 0 {
			fmt.Println("处理完成! 没有待分类的曲目! \n3秒后关闭此窗口...")
			time.Sleep(3 * time.Second)
			return
		}
		engine := gin.Default()
		engine.GET("/uncategorized", func(c *gin.Context) {
			marshal, err := json.Marshal(uncategorizedData)
			if err != nil {
				c.Writer.WriteString("json marshal failed: " + err.Error())
				return
			}
			c.Writer.Write(marshal)
		})
		go engine.Run(":" + strconv.Itoa(listenPort))
		fmt.Printf("处理完成! 请前往如下地址查看分类信息:\nhttp://127.0.0.1:%d/uncategorized\n\n\n\n", listenPort)
		select {}
	}
}
