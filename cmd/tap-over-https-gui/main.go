package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
	"github.com/zwh8800/tap-over-https/core"
)

//go:embed icon1.ico
var iconStopped []byte

//go:embed icon2.ico
var iconRunning []byte

type runStatus int

const (
	runStatusStopped runStatus = iota
	runStatusRunning
)

var (
	client *core.Client
	status = runStatusStopped
	addr   = "ws://www.baidu.com/vpn"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	var app MainApp
	app.OnReady()
}

func onExit() {

}

type MainApp struct {
	mRun    *systray.MenuItem
	mConfig *systray.MenuItem
	mQuit   *systray.MenuItem
}

func (m *MainApp) OnReady() {
	systray.SetIcon(iconStopped)
	m.mRun = systray.AddMenuItem("启动", "启动/停止vpn")
	m.mConfig = systray.AddMenuItem("配置地址", "配置vpn服务端地址，以ws或wss开头")
	m.mQuit = systray.AddMenuItem("退出", "退出程序")

	go m.onRunClick()
	go m.onConfigClick()
	go m.onQuitClick()
}

func (m *MainApp) runClient() {
	defer func() {
		err := recover()
		if err != nil {
			dlgs.Error("启动vpn时出错", fmt.Sprint(err))
		}
	}()

	client = core.NewClient(addr)
	client.Run()
	systray.SetIcon(iconRunning)
	m.mRun.SetTitle("停止")
}

func (m *MainApp) stopClient() {
	defer func() {
		err := recover()
		if err != nil {
			dlgs.Error("停止vpn时出错", fmt.Sprint(err))
		}
	}()
	client.Close()
	client = nil
	systray.SetIcon(iconStopped)
	m.mRun.SetTitle("启动")
}

func (m *MainApp) onRunClick() {
	for {
		<-m.mRun.ClickedCh
		if status == runStatusStopped {
			m.runClient()
			status = runStatusRunning
		} else {
			m.stopClient()
			status = runStatusStopped
		}

	}
}

func (m *MainApp) onConfigClick() {
	for {
		<-m.mConfig.ClickedCh
		input, ok, err := dlgs.Entry("配置地址", "请输入vpn地址", addr)
		if err != nil {
			panic(err)
		}
		if ok {
			addr = input
			if status == runStatusRunning {
				m.stopClient()
				m.runClient()
			}
		}
	}
}

func (m *MainApp) onQuitClick() {
	for {
		<-m.mQuit.ClickedCh
		os.Exit(0)
	}
}
