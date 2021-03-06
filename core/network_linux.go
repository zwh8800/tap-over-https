package core

import (
	"log"

	"github.com/vishvananda/netlink"
)

func createBridge(bridgeName, tapName string) {
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		log.Panicf("error on netlink.LinkByName: %s", err.Error())
	}

	tap, err := netlink.LinkByName(tapName)
	if err != nil {
		log.Panicf("netlink.LinkByName error: %s", err.Error())
	}
	err = netlink.LinkSetUp(tap)
	if err != nil {
		log.Panicf("netlink.LinkSetUp error: %s", err.Error())
	}
	err = netlink.LinkSetMaster(tap, bridge)
	if err != nil {
		log.Panicf("error on netlink.LinkSetMaster: %s", err.Error())
	}
}

func setupTapAddr(tapName string, ipBody *IPAssignBody) {
	ifaceLink, err := netlink.LinkByName(tapName)
	if err != nil {
		log.Panicf("netlink.LinkByName error: %s", err.Error())
	}
	addr, err := netlink.ParseAddr(ipBody.IP + "/24")
	if err != nil {
		log.Panicf("netlink.ParseAddr error: %s", err.Error())
	}
	err = netlink.AddrAdd(ifaceLink, addr)
	if err != nil {
		log.Panicf("netlink.AddrAdd error: %s", err.Error())
	}
	err = netlink.LinkSetUp(ifaceLink)
	if err != nil {
		log.Panicf("netlink.LinkSetUp error: %s", err.Error())
	}
}
