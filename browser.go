package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/mitchellh/go-ps"
)

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

// initBrowser 初始化浏览器参数
func initBrowser() {
	processes, err := ps.Processes()
	if err != nil {
		log.Fatal(err)
	}

	for _, process := range processes {
		executable := strings.ToLower(process.Executable())
		if strings.Contains(executable, "chrome") || strings.Contains(executable, "firefox") || strings.Contains(executable, "safari") || strings.Contains(executable, "msedge") {
			browserState = true
			defaultBrowser = strings.Split(executable, ".")[0]
			return
		}
	}
	browserState = false
	defaultBrowser = "chrome"
}

// closeBrowser 关闭浏览器
func closeBrowser() {
	var cmd *exec.Cmd
	cmd = exec.Command("taskkill", "/F", "/IM", defaultBrowser+".exe")
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("已关闭浏览器：%s\n", defaultBrowser)
}
