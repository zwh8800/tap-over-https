package core

import (
	"log"
	"net"
	"os/exec"
)

func createBridge(tapName string) {
	panic("not implement")
}

func setupTapAddr(tapName string, ipBody *IPAssignBody) {
	ip := net.ParseIP(ipBody.IP)
	if ip == nil {
		log.Panicf("assigned ip not valid: %s", ipBody.IP)
	}

	cmd := exec.Command("ifconfig", tapName, ip.To4().String()+"/24", "up")
	err := cmd.Run()
	if err != nil {
		log.Panicf("cmd.Run error: %s", err.Error())
	}
}
