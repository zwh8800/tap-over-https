package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
	"github.com/zwh8800/tap-over-https/core"
)

//go:embed icon1.ico
var iconStopped []byte

//go:embed icon2.ico
var iconRunning []byte

//go:embed icon-down.ico
var iconDown []byte

//go:embed icon-up.ico
var iconUp []byte

const configFileName = "taps"

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

type configFile struct {
	Addr string
}

func loadConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	data, err := ioutil.ReadFile(path.Join(home, ".config", configFileName))
	if err != nil {
		return
	}
	var conf configFile
	err = json.Unmarshal(data, &conf)
	if err != nil {
		return
	}

	addr = conf.Addr
}

func saveConfig() {
	var conf configFile
	conf.Addr = addr

	data, err := json.Marshal(&conf)
	if err != nil {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	configDir := path.Join(home, ".config")

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
	}

	err = ioutil.WriteFile(path.Join(configDir, configFileName), data, 0644)
	if err != nil {
		return
	}
}

func main() {
	loadConfig()
	systray.Run(onReady, onExit)
}

func onReady() {
	var app MainApp
	app.OnReady()
}

func onExit() {

}

type MainApp struct {
	mRun       *systray.MenuItem
	mConfig    *systray.MenuItem
	mSpeedUp   *systray.MenuItem
	mSpeedDown *systray.MenuItem
	mQuit      *systray.MenuItem
}

func (m *MainApp) OnReady() {
	systray.SetIcon(iconStopped)
	m.mRun = systray.AddMenuItem("启动", "启动/停止vpn")
	m.mConfig = systray.AddMenuItem("配置地址", "配置vpn服务端地址，以ws或wss开头")
	systray.AddSeparator()
	m.mSpeedUp = systray.AddMenuItem(" 0 kb/s", "")
	m.mSpeedUp.SetIcon(iconUp)
	m.mSpeedDown = systray.AddMenuItem(" 0 kb/s", "")
	m.mSpeedDown.SetIcon(iconDown)
	systray.AddSeparator()
	m.mQuit = systray.AddMenuItem("退出", "退出程序")

	go m.onRunClick()
	go m.onConfigClick()
	go m.onQuitClick()

	go m.syncStatus()
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
			saveConfig()
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

func (m *MainApp) syncStatus() {
	for {
		time.Sleep(1 * time.Second)
		m.syncSingle()
	}
}

func (m *MainApp) syncSingle() {
	defer func() {
		err := recover()
		if err != nil {
			log.Printf("panic on syncSingle: %#v", err)
		}
	}()
	if client == nil {
		return
	}
	if client.GetStatus() == core.ClientStatusRunning && status != runStatusRunning {
		systray.SetIcon(iconRunning)
		m.mRun.SetTitle("停止")
		status = runStatusRunning
	} else if client.GetStatus() == core.ClientStatusStopped && status != runStatusStopped {
		systray.SetIcon(iconStopped)
		m.mRun.SetTitle("启动")
		status = runStatusStopped
	}

	upSpeed, downSpeed := client.GetSpeed()
	m.mSpeedUp.SetTitle(fmt.Sprintf(" %s/s", humanize.Bytes(uint64(upSpeed))))
	m.mSpeedDown.SetTitle(fmt.Sprintf(" %s/s", humanize.Bytes(uint64(downSpeed))))
}
