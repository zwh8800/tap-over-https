package main

import (
	"log"

	"github.com/vishvananda/netlink"
)

func createBridge(tapName string) {
	la := netlink.NewLinkAttrs()
	la.Name = "br-tap"
	bridge := &netlink.Bridge{LinkAttrs: la}
	err := netlink.LinkAdd(bridge)
	if err != nil {
		log.Panicf("error on netlink.LinkAdd: %s", err.Error())
	}
	err = netlink.LinkSetUp(bridge)
	if err != nil {
		log.Panicf("error on netlink.LinkSetUp: %s", err.Error())
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

	eth, err := netlink.LinkByName(cmdIFaceBridge)
	if err != nil {
		log.Panicf("error on netlink.LinkByName: %s", err.Error())
	}

	addr, err := netlink.AddrList(eth, netlink.FAMILY_V4)
	if err != nil {
		log.Panicf("error on netlink.AddrList: %s", err.Error())
	}
	if len(addr) < 0 {
		log.Panicf("error on netlink.AddrList: addr empty")
	}
	err = netlink.AddrAdd(bridge, &addr[0])
	if err != nil {
		log.Panicf("error on netlink.AddrAdd: %s", err.Error())
	}
	err = netlink.AddrDel(eth, &addr[0])
	if err != nil {
		log.Panicf("error on netlink.AddrAdd: %s", err.Error())
	}
	err = netlink.LinkSetMaster(eth, bridge)
	if err != nil {
		log.Panicf("error on netlink.LinkSetMaster: %s", err.Error())
	}
}

func setupTapAddr(tapName string, ipBody *IPAssignBody) {
	ifaceLink, err := netlink.LinkByName(tapName)
	if err != nil {
		log.Panicf("netlink.LinkByName error: %s", err.Error())
	}
	addr, err := netlink.ParseAddr(ipBody.IP)
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
