package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zwh8800/tap-over-https/core"
)

var (
	cmdIsServer    bool
	cmdAddr        string
	cmdIpStart     string
	cmdIpEnd       string
	cmdIFaceBridge string
)

func main() {
	parseCmd()

	if cmdIsServer {
		server := core.NewServer(cmdAddr, cmdIpStart, cmdIpEnd, cmdIFaceBridge)
		server.Run()
	} else {
		client := core.NewClient(cmdAddr)
		client.Run()
		waitSignal(syscall.SIGINT, syscall.SIGTERM)
		client.Close()
		time.Sleep(5 * time.Second)
	}
}

func waitSignal(sig ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig...)
	<-c
}
func parseCmd() {
	flag.BoolVar(&cmdIsServer, "s", false, "run as server mode")
	flag.StringVar(&cmdAddr, "addr", ":8012", "vpn server address in client mode\nbind address in server mode")
	flag.StringVar(&cmdIpStart, "ip-start", "10.0.0.80", "ip start for client")
	flag.StringVar(&cmdIpEnd, "ip-end", "10.0.0.100", "ip end for client")
	flag.StringVar(&cmdIFaceBridge, "i", "br-lan", "bridge interface")
	flag.Parse()
}
