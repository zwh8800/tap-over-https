package main

import (
	"fmt"
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

	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf(`name="%s"`, tapName),
		"source=static",
		fmt.Sprintf("addr=%s", ip.To4().String()),
		"mask=255.255.255.0",
		"gateway=none")
	err := cmd.Run()
	if err != nil {
		log.Panicf("cmd.Run error: %s", err.Error())
	}
}
